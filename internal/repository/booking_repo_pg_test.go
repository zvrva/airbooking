package repository

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestNewBookingRepository(t *testing.T) {
	pool := &pgxpool.Pool{}
	repo := NewBookingRepository(pool)
	assert.NotNil(t, repo)
}