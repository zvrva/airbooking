package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Domenick1991/airbooking/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BookingRepository interface {
	CreatePending(ctx context.Context, booking *domain.Booking) error
	GetByToken(ctx context.Context, token string) (*domain.Booking, error)
	UpdateStatus(ctx context.Context, token string, status domain.BookingStatus) (*domain.Booking, error)
	ExpirePendingBefore(ctx context.Context, deadline time.Time) ([]domain.Booking, error)
	ReleaseSeat(ctx context.Context, flightID int64) error
}

type PGBookingRepository struct {
	db *pgxpool.Pool
}

func NewBookingRepository(db *pgxpool.Pool) BookingRepository {
	return &PGBookingRepository{db: db}
}

func (r *PGBookingRepository) CreatePending(ctx context.Context, booking *domain.Booking) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var available int
	if err := tx.QueryRow(ctx, `UPDATE flights SET available_seats = available_seats - 1, updated_at = now() WHERE id=$1 AND available_seats > 0 RETURNING available_seats`, booking.FlightID).Scan(&available); err != nil {
		return err
	}
	if available < 0 {
		return errors.New("no available seats")
	}

	booking.Status = domain.BookingStatusPending
	if err := tx.QueryRow(ctx, `INSERT INTO bookings (flight_id, seat_number, token, status, expires_at, email)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`, booking.FlightID, booking.SeatNumber, booking.Token, booking.Status, booking.ExpiresAt, booking.Email).
		Scan(&booking.ID, &booking.CreatedAt, &booking.UpdatedAt); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PGBookingRepository) GetByToken(ctx context.Context, token string) (*domain.Booking, error) {
	row := r.db.QueryRow(ctx, `SELECT id, flight_id, seat_number, token, status, expires_at, email, created_at, updated_at FROM bookings WHERE token=$1`, token)
	var b domain.Booking
	if err := row.Scan(&b.ID, &b.FlightID, &b.SeatNumber, &b.Token, &b.Status, &b.ExpiresAt, &b.Email, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *PGBookingRepository) UpdateStatus(ctx context.Context, token string, status domain.BookingStatus) (*domain.Booking, error) {
	row := r.db.QueryRow(ctx, `UPDATE bookings SET status=$1, updated_at=now() WHERE token=$2 RETURNING id, flight_id, seat_number, token, status, expires_at, email, created_at, updated_at`, status, token)
	var b domain.Booking
	if err := row.Scan(&b.ID, &b.FlightID, &b.SeatNumber, &b.Token, &b.Status, &b.ExpiresAt, &b.Email, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *PGBookingRepository) ExpirePendingBefore(ctx context.Context, deadline time.Time) ([]domain.Booking, error) {
	rows, err := r.db.Query(ctx, `UPDATE bookings SET status=$1, updated_at=now() WHERE status=$2 AND expires_at <= $3 RETURNING id, flight_id, seat_number, token, status, expires_at, email, created_at, updated_at`, domain.BookingStatusExpired, domain.BookingStatusPending, deadline)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expired []domain.Booking
	for rows.Next() {
		var b domain.Booking
		if err := rows.Scan(&b.ID, &b.FlightID, &b.SeatNumber, &b.Token, &b.Status, &b.ExpiresAt, &b.Email, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		expired = append(expired, b)
	}
	return expired, rows.Err()
}

func (r *PGBookingRepository) ReleaseSeat(ctx context.Context, flightID int64) error {
	cmd, err := r.db.Exec(ctx, `
        UPDATE flights 
        SET available_seats = available_seats + 1, 
            updated_at = now() 
        WHERE id = $1
    `, flightID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errors.New("flight not found")
	}

	_, err = r.db.Exec(ctx, `
        DELETE FROM bookings 
        WHERE flight_id = $1 
        AND status = 'CANCELLED'
    `, flightID)
	if err != nil {
		return err
	}

	return err
}

var _ BookingRepository = (*PGBookingRepository)(nil)
