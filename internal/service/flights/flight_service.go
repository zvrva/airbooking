package flights

import (
	"context"
	"time"

	"github.com/Domenick1991/airbooking/internal/domain"
	"github.com/Domenick1991/airbooking/internal/repository"
	"github.com/Domenick1991/airbooking/internal/service/booking"
)

type FlightUseCase interface {
	List(ctx context.Context) ([]domain.Flight, error)
	GetByID(ctx context.Context, id int64) (*domain.Flight, error)
}

type FlightService struct {
	repo     repository.FlightRepository
	cache    booking.Cache // Указатель на структуру
	cacheTTL time.Duration
}

// Интерфейс Cache (можно вынести в отдельный файл или оставить здесь)
// Он уже определен в booking_service.go, но можно продублировать здесь
// Лучше вынести в общий файл, но для простоты оставим здесь
type FlightCache interface {
	GetFlights(ctx context.Context) ([]domain.Flight, error)
	SetFlights(ctx context.Context, flights []domain.Flight) error
	AcquireSeatLock(ctx context.Context, flightID int64, seatNumber int, ttl time.Duration) (bool, error)
	ReleaseSeatLock(ctx context.Context, flightID int64, seatNumber int) error
}

// Оригинальный конструктор
func NewFlightService(repo repository.FlightRepository, cache FlightCache, cacheTTL time.Duration) *FlightService {
	return &FlightService{repo: repo, cache: cache, cacheTTL: cacheTTL}
}

func (s *FlightService) List(ctx context.Context) ([]domain.Flight, error) {
	if s.cache != nil {
		if cached, err := s.cache.GetFlights(ctx); err == nil && cached != nil {
			return cached, nil
		}
	}

	flights, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	if s.cache != nil {
		_ = s.cache.SetFlights(ctx, flights)
	}
	return flights, nil
}

func (s *FlightService) GetByID(ctx context.Context, id int64) (*domain.Flight, error) {
	return s.repo.GetByID(ctx, id)
}

var _ FlightUseCase = (*FlightService)(nil)
