package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Domenick1991/airbooking/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockFlightUseCase is a mock implementation of service.FlightUseCase
type MockFlightUseCase struct {
	mock.Mock
}

func (m *MockFlightUseCase) List(ctx context.Context) ([]domain.Flight, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.Flight), args.Error(1)
}

func (m *MockFlightUseCase) GetByID(ctx context.Context, id int64) (*domain.Flight, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Flight), args.Error(1)
}

func TestFlightHandler_list(t *testing.T) {
	mockService := &MockFlightUseCase{}
	handler := NewFlightHandler(mockService)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/flights", nil)

	flights := []domain.Flight{
		{ID: 1, FromAirport: "SVO", ToAirport: "LED", TotalSeats: 100, AvailableSeats: 50, PriceCents: 5000},
	}

	mockService.On("List", c.Request.Context()).Return(flights, nil)

	handler.list(c)

	assert.Equal(t, http.StatusOK, w.Code)
	// For simplicity, not parsing JSON, but in real test, check body

	mockService.AssertExpectations(t)
}

func TestFlightHandler_get(t *testing.T) {
	mockService := &MockFlightUseCase{}
	handler := NewFlightHandler(mockService)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest("GET", "/flights/1", nil)

	flight := &domain.Flight{
		ID: 1, FromAirport: "SVO", ToAirport: "LED", TotalSeats: 100, AvailableSeats: 50, PriceCents: 5000,
	}

	mockService.On("GetByID", c.Request.Context(), int64(1)).Return(flight, nil)

	handler.get(c)

	assert.Equal(t, http.StatusOK, w.Code)

	mockService.AssertExpectations(t)
}