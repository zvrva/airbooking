package booking

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Domenick1991/airbooking/internal/domain"
	"github.com/Domenick1991/airbooking/internal/kafka"
	"github.com/Domenick1991/airbooking/internal/repository"
	"github.com/google/uuid"
)

type BookingUseCase interface {
	CreateBooking(ctx context.Context, input CreateBookingInput) (*domain.Booking, error)
	ConfirmBooking(ctx context.Context, token string) (*domain.Booking, error)
	CancelBooking(ctx context.Context, token string) (*domain.Booking, error)
	ExpirePendingBookings(ctx context.Context) ([]domain.Booking, error)
}

type Cache interface {
	AcquireSeatLock(ctx context.Context, flightID int64, seatNumber int, ttl time.Duration) (bool, error)
	ReleaseSeatLock(ctx context.Context, flightID int64, seatNumber int) error
	GetFlights(ctx context.Context) ([]domain.Flight, error)
	SetFlights(ctx context.Context, flights []domain.Flight) error
}

type Producer interface {
	Publish(ctx context.Context, topic, key string, value interface{}) error
}

type BookingService struct {
	bookings           repository.BookingRepository
	flights            repository.FlightRepository
	cache              Cache    // Указатель на структуру
	producer           Producer // Указатель на структуру
	bookingTopic       string
	notificationsTopic string
	holdTTL            time.Duration
	confirmationTTL    time.Duration
}

type CreateBookingInput struct {
	FlightID   int64  `json:"flight_id"`
	SeatNumber int    `json:"seat_number"`
	Email      string `json:"email"`
}
type BookingServiceOption func(*BookingService)

// Интерфейсы для тестирования (оставляем в том же пакете)

type EventProducer interface {
	Publish(ctx context.Context, topic, key string, value interface{}) error
}

func WithNotificationsTopic(topic string) BookingServiceOption {
	return func(s *BookingService) {
		s.notificationsTopic = topic
	}
}

// Оригинальный конструктор
func NewBookingService(
	bookings repository.BookingRepository,
	flights repository.FlightRepository,
	cache Cache,
	producer *kafka.Producer,
	bookingTopic string,
	holdTTL, confirmationTTL time.Duration,
	opts ...BookingServiceOption,
) *BookingService {
	service := &BookingService{
		bookings:        bookings,
		flights:         flights,
		cache:           cache,
		producer:        producer,
		bookingTopic:    bookingTopic,
		holdTTL:         holdTTL,
		confirmationTTL: confirmationTTL,
	}
	for _, opt := range opts {
		opt(service)
	}
	return service
}

func (s *BookingService) CreateBooking(ctx context.Context, input CreateBookingInput) (*domain.Booking, error) {
	if input.SeatNumber <= 0 {
		return nil, errors.New("seat number must be positive")
	}
	if input.Email == "" {
		return nil, errors.New("email is required")
	}

	log.Printf("Cache interface: %v", s.cache)
	log.Printf("Cache is nil: %v", s.cache == nil)

	locked := false
	if s.cache != nil {
		log.Println("Cache is not nil, attempting to acquire seat lock...")
		ok, err := s.cache.AcquireSeatLock(ctx, input.FlightID, input.SeatNumber, s.holdTTL)
		if err != nil {
			log.Printf("Error acquiring seat lock: %v", err)
			return nil, err
		}
		if !ok {
			log.Println("Seat is already locked")
			return nil, errors.New("seat is already locked")
		}
		locked = true
	} else {
		log.Println("Cache is nil, skipping lock acquisition")
	}

	expiresIn := s.confirmationTTL
	if expiresIn == 0 {
		expiresIn = s.holdTTL
	}

	booking := &domain.Booking{
		FlightID:   input.FlightID,
		SeatNumber: input.SeatNumber,
		Token:      uuid.NewString(),
		ExpiresAt:  time.Now().Add(expiresIn),
		Email:      input.Email,
	}

	if err := s.bookings.CreatePending(ctx, booking); err != nil {
		if locked {
			_ = s.cache.ReleaseSeatLock(ctx, input.FlightID, input.SeatNumber)
		}
		return nil, err
	}

	booking.Status = domain.BookingStatusPending
	if err := s.publish(ctx, "booking_created", booking); err != nil {
		fmt.Printf("WARNING: Failed to publish booking_created event for booking %s: %v\n", booking.Token, err)
	}
	return booking, nil
}

func (s *BookingService) ConfirmBooking(ctx context.Context, token string) (*domain.Booking, error) {
	current, err := s.bookings.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if current.Status != domain.BookingStatusPending {
		return nil, errors.New("booking is not pending")
	}

	updated, err := s.bookings.UpdateStatus(ctx, token, domain.BookingStatusConfirmed)
	if err != nil {
		return nil, err
	}
	if err := s.publish(ctx, "booking_confirmed", updated); err != nil {
		fmt.Printf("WARNING: Failed to publish booking_confirmed event for booking %s: %v\n", updated.Token, err)
	}
	if s.cache != nil {
		_ = s.cache.ReleaseSeatLock(ctx, updated.FlightID, updated.SeatNumber)
	}
	return updated, nil
}

func (s *BookingService) CancelBooking(ctx context.Context, token string) (*domain.Booking, error) {
	current, err := s.bookings.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if current.Status == domain.BookingStatusCancelled || current.Status == domain.BookingStatusExpired {
		return current, nil
	}

	updated, err := s.bookings.UpdateStatus(ctx, token, domain.BookingStatusCancelled)
	if err != nil {
		return nil, err
	}
	_ = s.bookings.ReleaseSeat(ctx, updated.FlightID)
	if err := s.publish(ctx, "booking_cancelled", updated); err != nil {
		fmt.Printf("WARNING: Failed to publish booking_cancelled event for booking %s: %v\n", updated.Token, err)
	}
	if s.cache != nil {
		_ = s.cache.ReleaseSeatLock(ctx, updated.FlightID, updated.SeatNumber)
	}
	return updated, nil
}

func (s *BookingService) ExpirePendingBookings(ctx context.Context) ([]domain.Booking, error) {
	deadline := time.Now()
	expired, err := s.bookings.ExpirePendingBefore(ctx, deadline)
	if err != nil {
		return nil, err
	}
	for _, b := range expired {
		_ = s.bookings.ReleaseSeat(ctx, b.FlightID)
		_ = s.publish(ctx, "booking_expired", &b)
		if s.cache != nil {
			_ = s.cache.ReleaseSeatLock(ctx, b.FlightID, b.SeatNumber)
		}
	}
	return expired, nil
}

func (s *BookingService) publish(ctx context.Context, eventType string, booking *domain.Booking) error {
	if s.producer == nil || s.bookingTopic == "" {
		return nil
	}
	event := kafka.BookingEvent{
		Type:       eventType,
		Token:      booking.Token,
		FlightID:   booking.FlightID,
		SeatNumber: booking.SeatNumber,
		Email:      booking.Email,
		Status:     string(booking.Status),
		ExpiresAt:  booking.ExpiresAt,
	}
	if err := s.producer.Publish(ctx, s.bookingTopic, booking.Token, event); err != nil {
		return err
	}
	if s.notificationsTopic != "" {
		return s.producer.Publish(ctx, s.notificationsTopic, booking.Token, event)
	}
	return nil
}

var _ BookingUseCase = (*BookingService)(nil)
