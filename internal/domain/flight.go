package domain

import "time"

type Flight struct {
	ID             int64
	FromAirport    string
	ToAirport      string
	DepartureTime  time.Time
	ArrivalTime    time.Time
	TotalSeats     int
	AvailableSeats int
	PriceCents     int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
