package flights

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Domenick1991/airbooking/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockFlightRepository struct {
	mock.Mock
}

func (m *MockFlightRepository) List(ctx context.Context) ([]domain.Flight, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.Flight), args.Error(1)
}

func (m *MockFlightRepository) GetByID(ctx context.Context, id int64) (*domain.Flight, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Flight), args.Error(1)
}

func (m *MockFlightRepository) ReserveSeat(ctx context.Context, flightID int64) error {
	args := m.Called(ctx, flightID)
	return args.Error(0)
}

func (m *MockFlightRepository) ReleaseSeat(ctx context.Context, flightID int64) error {
	args := m.Called(ctx, flightID)
	return args.Error(0)
}

type MockCache struct {
	mock.Mock
}

func (m *MockCache) AcquireSeatLock(ctx context.Context, flightID int64, seatNumber int, ttl time.Duration) (bool, error) {
	args := m.Called(ctx, flightID, seatNumber, ttl)
	return args.Bool(0), args.Error(1)
}

func (m *MockCache) ReleaseSeatLock(ctx context.Context, flightID int64, seatNumber int) error {
	args := m.Called(ctx, flightID, seatNumber)
	return args.Error(0)
}

func (m *MockCache) GetFlights(ctx context.Context) ([]domain.Flight, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.Flight), args.Error(1)
}

func (m *MockCache) SetFlights(ctx context.Context, flights []domain.Flight) error {
	args := m.Called(ctx, flights)
	return args.Error(0)
}

func TestFlightService_List_CacheMiss(t *testing.T) {
	mockRepo := &MockFlightRepository{}
	mockCache := &MockCache{}

	service := NewFlightService(mockRepo, mockCache, time.Minute)

	ctx := context.Background()

	flights := []domain.Flight{
		{
			ID:             4,
			FromAirport:    "SVO",
			ToAirport:      "LED",
			DepartureTime:  time.Now(),
			ArrivalTime:    time.Now().Add(time.Hour),
			TotalSeats:     150,
			AvailableSeats: 149,
			PriceCents:     500000,
		},
	}

	// Кэш пустой
	mockCache.On("GetFlights", ctx).Return(([]domain.Flight)(nil), nil).Once()
	mockRepo.On("List", ctx).Return(flights, nil).Once()
	mockCache.On("SetFlights", ctx, flights).Return(nil).Once()

	result, err := service.List(ctx)

	assert.NoError(t, err)
	assert.Equal(t, flights, result)

	mockCache.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

// Тест 21: Получение списка рейсов - данные в кэше
func TestFlightService_List_CacheHit(t *testing.T) {
	mockRepo := &MockFlightRepository{}
	mockCache := &MockCache{}

	service := NewFlightService(mockRepo, mockCache, time.Minute)

	ctx := context.Background()

	flights := []domain.Flight{
		{
			ID:             4,
			FromAirport:    "SVO",
			ToAirport:      "LED",
			DepartureTime:  time.Now(),
			ArrivalTime:    time.Now().Add(time.Hour),
			TotalSeats:     150,
			AvailableSeats: 149,
			PriceCents:     500000,
		},
	}

	// Данные есть в кэше
	mockCache.On("GetFlights", ctx).Return(flights, nil).Once()

	result, err := service.List(ctx)

	assert.NoError(t, err)
	assert.Equal(t, flights, result)

	mockCache.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "List")
	mockCache.AssertNotCalled(t, "SetFlights")
}

// Тест 22: Получение списка рейсов - ошибка в кэше
func TestFlightService_List_CacheError(t *testing.T) {
	mockRepo := &MockFlightRepository{}
	mockCache := &MockCache{}

	service := NewFlightService(mockRepo, mockCache, time.Minute)

	ctx := context.Background()

	flights := []domain.Flight{
		{
			ID:             4,
			FromAirport:    "SVO",
			ToAirport:      "LED",
			DepartureTime:  time.Now(),
			ArrivalTime:    time.Now().Add(time.Hour),
			TotalSeats:     150,
			AvailableSeats: 149,
			PriceCents:     500000,
		},
	}

	// Ошибка при получении из кэша
	mockCache.On("GetFlights", ctx).Return(([]domain.Flight)(nil), errors.New("cache error")).Once()
	mockRepo.On("List", ctx).Return(flights, nil).Once()
	mockCache.On("SetFlights", ctx, flights).Return(nil).Once()

	result, err := service.List(ctx)

	assert.NoError(t, err)
	assert.Equal(t, flights, result)

	mockCache.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

// Тест 23: Получение списка рейсов - ошибка в репозитории
func TestFlightService_List_RepositoryError(t *testing.T) {
	mockRepo := &MockFlightRepository{}
	mockCache := &MockCache{}

	service := NewFlightService(mockRepo, mockCache, time.Minute)

	ctx := context.Background()

	// Ошибка в репозитории
	expectedErr := errors.New("database error")
	mockCache.On("GetFlights", ctx).Return(([]domain.Flight)(nil), nil).Once()
	mockRepo.On("List", ctx).Return([]domain.Flight{}, expectedErr).Once()

	result, err := service.List(ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedErr, err)

	mockCache.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockCache.AssertNotCalled(t, "SetFlights")
}

// Тест 24: Получение рейса по ID
func TestFlightService_GetByID_Success(t *testing.T) {
	mockRepo := &MockFlightRepository{}
	mockCache := &MockCache{}

	service := NewFlightService(mockRepo, mockCache, time.Minute)

	ctx := context.Background()
	id := int64(4)

	flight := &domain.Flight{
		ID:             4,
		FromAirport:    "SVO",
		ToAirport:      "LED",
		DepartureTime:  time.Now(),
		ArrivalTime:    time.Now().Add(time.Hour),
		TotalSeats:     150,
		AvailableSeats: 149,
		PriceCents:     500000,
	}

	mockRepo.On("GetByID", ctx, id).Return(flight, nil).Once()

	result, err := service.GetByID(ctx, id)

	assert.NoError(t, err)
	assert.Equal(t, flight, result)

	mockRepo.AssertExpectations(t)
}

// Тест 25: Получение рейса по ID - не найден
func TestFlightService_GetByID_NotFound(t *testing.T) {
	mockRepo := &MockFlightRepository{}
	mockCache := &MockCache{}

	service := NewFlightService(mockRepo, mockCache, time.Minute)

	ctx := context.Background()
	id := int64(999)

	// Рейс не найден
	expectedErr := errors.New("flight not found")
	mockRepo.On("GetByID", ctx, id).Return(nil, expectedErr).Once()

	result, err := service.GetByID(ctx, id)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedErr, err)

	mockRepo.AssertExpectations(t)
}

// Тест 31: Тест работы с кэшем без кэша
func TestFlightService_NoCache(t *testing.T) {
	mockRepo := &MockFlightRepository{}

	service := NewFlightService(mockRepo, nil, time.Minute)

	ctx := context.Background()

	flights := []domain.Flight{
		{
			ID:             4,
			FromAirport:    "SVO",
			ToAirport:      "LED",
			DepartureTime:  time.Now(),
			ArrivalTime:    time.Now().Add(time.Hour),
			TotalSeats:     150,
			AvailableSeats: 149,
			PriceCents:     500000,
		},
	}

	// Должен вызываться только репозиторий
	mockRepo.On("List", ctx).Return(flights, nil).Once()

	result, err := service.List(ctx)

	assert.NoError(t, err)
	assert.Equal(t, flights, result)

	mockRepo.AssertExpectations(t)
}
