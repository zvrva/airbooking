package email

import (
	"context"
	"fmt"

	"github.com/Domenick1991/airbooking/internal/kafka"
)

type Sender struct{}

func NewSender() *Sender {
	return &Sender{}
}

func (s *Sender) Send(ctx context.Context, event kafka.BookingEvent) error {
	fmt.Printf("send email to %s about %s for flight %d seat %d\n", event.Email, event.Type, event.FlightID, event.SeatNumber)
	return nil
}
