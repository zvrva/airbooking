package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Domenick1991/airbooking/internal/domain"
	"github.com/Domenick1991/airbooking/internal/service/booking"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBookingUseCase is a mock implementation of service.BookingUseCase
type MockBookingUseCase struct {
	mock.Mock
}

func (m *MockBookingUseCase) CreateBooking(ctx context.Context, input booking.CreateBookingInput) (*domain.Booking, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Booking), args.Error(1)
}

func (m *MockBookingUseCase) ConfirmBooking(ctx context.Context, token string) (*domain.Booking, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Booking), args.Error(1)
}

func (m *MockBookingUseCase) CancelBooking(ctx context.Context, token string) (*domain.Booking, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Booking), args.Error(1)
}

func (m *MockBookingUseCase) ExpirePendingBookings(ctx context.Context) ([]domain.Booking, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.Booking), args.Error(1)
}

func TestBookingHandler_create(t *testing.T) {
	mockService := &MockBookingUseCase{}
	handler := NewBookingHandler(mockService)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	input := booking.CreateBookingInput{
		FlightID:   1,
		SeatNumber: 10,
		Email:      "test@example.com",
	}
	body, _ := json.Marshal(input)
	c.Request = httptest.NewRequest("POST", "/bookings", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	booking := &domain.Booking{
		ID:         1,
		FlightID:   1,
		SeatNumber: 10,
		Token:      "token123",
		Status:     domain.BookingStatusPending,
		Email:      "test@example.com",
	}

	mockService.On("CreateBooking", c.Request.Context(), input).Return(booking, nil)

	handler.create(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response bookingResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "token123", response.Token)
	assert.Equal(t, string(domain.BookingStatusPending), response.Status)

	mockService.AssertExpectations(t)
}

func TestBookingHandler_confirm(t *testing.T) {
	mockService := &MockBookingUseCase{}
	handler := NewBookingHandler(mockService)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	token := "token123"
	c.Params = gin.Params{{Key: "token", Value: token}}
	c.Request = httptest.NewRequest("PUT", "/bookings/"+token, nil)

	booking := &domain.Booking{
		ID:         1,
		FlightID:   1,
		SeatNumber: 10,
		Token:      token,
		Status:     domain.BookingStatusConfirmed,
		Email:      "test@example.com",
	}

	mockService.On("ConfirmBooking", c.Request.Context(), token).Return(booking, nil)

	handler.confirm(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response bookingResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, string(domain.BookingStatusConfirmed), response.Status)

	mockService.AssertExpectations(t)
}

func TestBookingHandler_cancel(t *testing.T) {
	mockService := &MockBookingUseCase{}
	handler := NewBookingHandler(mockService)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	token := "token123"
	c.Params = gin.Params{{Key: "token", Value: token}}
	c.Request = httptest.NewRequest("DELETE", "/bookings/"+token, nil)

	booking := &domain.Booking{
		ID:         1,
		FlightID:   1,
		SeatNumber: 10,
		Token:      token,
		Status:     domain.BookingStatusCancelled,
		Email:      "test@example.com",
	}

	mockService.On("CancelBooking", c.Request.Context(), token).Return(booking, nil)

	handler.cancel(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response bookingResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, string(domain.BookingStatusCancelled), response.Status)

	mockService.AssertExpectations(t)
}
