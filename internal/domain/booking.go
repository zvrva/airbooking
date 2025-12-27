package domain

import "time"

type BookingStatus string

const (
	BookingStatusPending   BookingStatus = "PENDING"
	BookingStatusConfirmed BookingStatus = "CONFIRMED"
	BookingStatusCancelled BookingStatus = "CANCELLED"
	BookingStatusExpired   BookingStatus = "EXPIRED"
)

type Booking struct {
	ID         int64
	FlightID   int64
	SeatNumber int
	Token      string
	Status     BookingStatus
	ExpiresAt  time.Time
	Email      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
