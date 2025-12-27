package flights_service_api

import (
	"context"
	"time"

	"github.com/Domenick1991/airbooking/internal/domain"
	"github.com/Domenick1991/airbooking/internal/pb/flights_api"
	"github.com/Domenick1991/airbooking/internal/pb/models"
	"github.com/Domenick1991/airbooking/internal/service/flights"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Server implements the generated gRPC interface for flights.
type Server struct {
	flights flights.FlightUseCase
	flights_api.UnimplementedFlightsServiceServer
}

func NewServer(flights flights.FlightUseCase) *Server {
	return &Server{flights: flights}
}

func (s *Server) ListFlights(ctx context.Context, _ *emptypb.Empty) (*flights_api.ListFlightsResponse, error) {
	list, err := s.flights.List(ctx)
	if err != nil {
		return nil, err
	}
	resp := &flights_api.ListFlightsResponse{
		Flights: make([]*models.Flight, 0, len(list)),
	}
	for _, f := range list {
		resp.Flights = append(resp.Flights, toPBFlight(&f))
	}
	return resp, nil
}

func (s *Server) GetFlight(ctx context.Context, req *flights_api.GetFlightRequest) (*flights_api.GetFlightResponse, error) {
	flight, err := s.flights.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &flights_api.GetFlightResponse{Flight: toPBFlight(flight)}, nil
}

func toPBFlight(f *domain.Flight) *models.Flight {
	if f == nil {
		return nil
	}
	return &models.Flight{
		Id:             f.ID,
		FromAirport:    f.FromAirport,
		ToAirport:      f.ToAirport,
		DepartureTime:  f.DepartureTime.Format(time.RFC3339),
		ArrivalTime:    f.ArrivalTime.Format(time.RFC3339),
		TotalSeats:     int32(f.TotalSeats),
		AvailableSeats: int32(f.AvailableSeats),
		PriceCents:     f.PriceCents,
	}
}
