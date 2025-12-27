package repository

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestNewFlightRepository(t *testing.T) {
	pool := &pgxpool.Pool{} 
	repo := NewFlightRepository(pool)
	assert.NotNil(t, repo)
}
