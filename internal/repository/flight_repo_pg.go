package repository

import (
	"context"
	"errors"

	"github.com/Domenick1991/airbooking/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FlightRepository interface {
	List(ctx context.Context) ([]domain.Flight, error)
	GetByID(ctx context.Context, id int64) (*domain.Flight, error)
	ReserveSeat(ctx context.Context, flightID int64) error
	ReleaseSeat(ctx context.Context, flightID int64) error
}

type PGFlightRepository struct {
	db *pgxpool.Pool
}

func NewFlightRepository(db *pgxpool.Pool) FlightRepository {
	return &PGFlightRepository{db: db}
}

func (r *PGFlightRepository) List(ctx context.Context) ([]domain.Flight, error) {
	rows, err := r.db.Query(ctx, `SELECT id, from_airport, to_airport, departure_time, arrival_time, total_seats, available_seats, price_cents, created_at, updated_at FROM flights ORDER BY departure_time`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	flights := make([]domain.Flight, 0)
	for rows.Next() {
		var f domain.Flight
		if err := rows.Scan(&f.ID, &f.FromAirport, &f.ToAirport, &f.DepartureTime, &f.ArrivalTime, &f.TotalSeats, &f.AvailableSeats, &f.PriceCents, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		flights = append(flights, f)
	}
	return flights, rows.Err()
}

func (r *PGFlightRepository) GetByID(ctx context.Context, id int64) (*domain.Flight, error) {
	row := r.db.QueryRow(ctx, `SELECT id, from_airport, to_airport, departure_time, arrival_time, total_seats, available_seats, price_cents, created_at, updated_at FROM flights WHERE id=$1`, id)
	var f domain.Flight
	if err := row.Scan(&f.ID, &f.FromAirport, &f.ToAirport, &f.DepartureTime, &f.ArrivalTime, &f.TotalSeats, &f.AvailableSeats, &f.PriceCents, &f.CreatedAt, &f.UpdatedAt); err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *PGFlightRepository) ReserveSeat(ctx context.Context, flightID int64) error {
	res, err := r.db.Exec(ctx, `UPDATE flights SET available_seats = available_seats - 1, updated_at = now() WHERE id=$1 AND available_seats > 0`, flightID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New("no available seats")
	}
	return nil
}

func (r *PGFlightRepository) ReleaseSeat(ctx context.Context, flightID int64) error {
	_, err := r.db.Exec(ctx, `UPDATE flights SET available_seats = available_seats + 1, updated_at = now() WHERE id=$1`, flightID)
	return err
}

var _ FlightRepository = (*PGFlightRepository)(nil)
