package bookings_service_api

import (
	"context"
	"time"

	"github.com/Domenick1991/airbooking/internal/domain"
	"github.com/Domenick1991/airbooking/internal/pb/bookings_api"
	"github.com/Domenick1991/airbooking/internal/pb/models"
	"github.com/Domenick1991/airbooking/internal/service/booking"
)

// Server implements the generated gRPC interface for bookings.
type Server struct {
	bookings booking.BookingUseCase
	bookings_api.UnimplementedBookingsServiceServer
}

func NewServer(bookings booking.BookingUseCase) *Server {
	return &Server{bookings: bookings}
}

func (s *Server) CreateBooking(ctx context.Context, req *bookings_api.CreateBookingRequest) (*models.Booking, error) {
	created, err := s.bookings.CreateBooking(ctx, booking.CreateBookingInput{
		FlightID:   req.GetFlightId(),
		SeatNumber: int(req.GetSeatNumber()),
		Email:      req.GetEmail(),
	})
	if err != nil {
		return nil, err
	}
	return toPBBooking(created), nil
}

func (s *Server) ConfirmBooking(ctx context.Context, req *bookings_api.BookingTokenRequest) (*models.Booking, error) {
	booking, err := s.bookings.ConfirmBooking(ctx, req.GetToken())
	if err != nil {
		return nil, err
	}
	return toPBBooking(booking), nil
}

func (s *Server) CancelBooking(ctx context.Context, req *bookings_api.BookingTokenRequest) (*models.Booking, error) {
	booking, err := s.bookings.CancelBooking(ctx, req.GetToken())
	if err != nil {
		return nil, err
	}
	return toPBBooking(booking), nil
}

func toPBBooking(b *domain.Booking) *models.Booking {
	if b == nil {
		return nil
	}

	return &models.Booking{
		Token:      b.Token,
		Status:     toPBStatus(b.Status),
		ExpiresAt:  b.ExpiresAt.Format(time.RFC3339),
		FlightId:   b.FlightID,
		SeatNumber: int32(b.SeatNumber),
		Email:      b.Email,
	}
}

func toPBStatus(status domain.BookingStatus) models.BookingStatus {
	switch status {
	case domain.BookingStatusPending:
		return models.BookingStatus_BOOKING_STATUS_PENDING
	case domain.BookingStatusConfirmed:
		return models.BookingStatus_BOOKING_STATUS_CONFIRMED
	case domain.BookingStatusCancelled:
		return models.BookingStatus_BOOKING_STATUS_CANCELLED
	case domain.BookingStatusExpired:
		return models.BookingStatus_BOOKING_STATUS_EXPIRED
	default:
		return models.BookingStatus_BOOKING_STATUS_UNSPECIFIED
	}
}
