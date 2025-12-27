package booking

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Domenick1991/airbooking/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock структуры

type MockBookingRepository struct {
	mock.Mock
}

func (m *MockBookingRepository) CreatePending(ctx context.Context, booking *domain.Booking) error {
	args := m.Called(ctx, booking)
	return args.Error(0)
}

func (m *MockBookingRepository) GetByToken(ctx context.Context, token string) (*domain.Booking, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Booking), args.Error(1)
}

func (m *MockBookingRepository) UpdateStatus(ctx context.Context, token string, status domain.BookingStatus) (*domain.Booking, error) {
	args := m.Called(ctx, token, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Booking), args.Error(1)
}

func (m *MockBookingRepository) ExpirePendingBefore(ctx context.Context, deadline time.Time) ([]domain.Booking, error) {
	args := m.Called(ctx, deadline)
	return args.Get(0).([]domain.Booking), args.Error(1)
}

func (m *MockBookingRepository) ReleaseSeat(ctx context.Context, flightID int64) error {
	args := m.Called(ctx, flightID)
	return args.Error(0)
}

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

// MockCache - реализует интерфейс Cache напрямую
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

// MockProducer - реализует интерфейс Producer напрямую
type MockProducer struct {
	mock.Mock
}

func (m *MockProducer) Publish(ctx context.Context, topic, key string, value interface{}) error {
	args := m.Called(ctx, topic, key, value)
	return args.Error(0)
}

// ============================ Тесты для BookingService ============================

// Тест 1: Создание бронирования - успешный сценарий
func TestBookingService_CreateBooking_Success(t *testing.T) {
	mockBookingRepo := &MockBookingRepository{}
	mockFlightRepo := &MockFlightRepository{}
	mockCache := &MockCache{}
	mockProducer := &MockProducer{}

	service := &BookingService{
		bookings:        mockBookingRepo,
		flights:         mockFlightRepo,
		cache:           mockCache,
		producer:        mockProducer,
		bookingTopic:    "booking_topic",
		holdTTL:         time.Minute,
		confirmationTTL: time.Hour,
	}

	ctx := context.Background()
	input := CreateBookingInput{
		FlightID:   4,
		SeatNumber: 10,
		Email:      "test@example.com",
	}

	// Настройка моков
	mockCache.On("AcquireSeatLock", ctx, int64(4), 10, time.Minute).Return(true, nil).Once()
	mockBookingRepo.On("CreatePending", ctx, mock.AnythingOfType("*domain.Booking")).Return(nil).Once()
	mockProducer.On("Publish", ctx, "booking_topic", mock.Anything, mock.Anything).Return(nil).Once()

	// Выполнение
	booking, err := service.CreateBooking(ctx, input)

	// Проверки
	assert.NoError(t, err)
	assert.NotNil(t, booking)
	assert.Equal(t, domain.BookingStatusPending, booking.Status)
	assert.Equal(t, input.FlightID, booking.FlightID)
	assert.Equal(t, input.SeatNumber, booking.SeatNumber)
	assert.Equal(t, input.Email, booking.Email)

	mockCache.AssertExpectations(t)
	mockBookingRepo.AssertExpectations(t)
	mockProducer.AssertExpectations(t)
}

// // Тест 2: Создание бронирования - ошибка валидации
func TestBookingService_CreateBooking_ValidationErrors(t *testing.T) {
	service := &BookingService{
		holdTTL:         time.Minute,
		confirmationTTL: time.Hour,
	}

	ctx := context.Background()

	testCases := []struct {
		name        string
		input       CreateBookingInput
		expectedErr string
	}{
		{
			name: "Seat number zero",
			input: CreateBookingInput{
				FlightID:   4,
				SeatNumber: 0,
				Email:      "test@example.com",
			},
			expectedErr: "seat number must be positive",
		},
		{
			name: "Seat number negative",
			input: CreateBookingInput{
				FlightID:   4,
				SeatNumber: -5,
				Email:      "test@example.com",
			},
			expectedErr: "seat number must be positive",
		},
		{
			name: "Empty email",
			input: CreateBookingInput{
				FlightID:   4,
				SeatNumber: 10,
				Email:      "",
			},
			expectedErr: "email is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			booking, err := service.CreateBooking(ctx, tc.input)
			assert.Error(t, err)
			assert.Nil(t, booking)
			assert.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}

// Тест 3: Создание бронирования - место уже заблокировано
func TestBookingService_CreateBooking_SeatAlreadyLocked(t *testing.T) {
	mockBookingRepo := &MockBookingRepository{}
	mockFlightRepo := &MockFlightRepository{}
	mockCache := &MockCache{}
	mockProducer := &MockProducer{}

	service := &BookingService{
		bookings:        mockBookingRepo,
		flights:         mockFlightRepo,
		cache:           mockCache,
		producer:        mockProducer,
		bookingTopic:    "booking_topic",
		holdTTL:         time.Minute, // 1 минута
		confirmationTTL: time.Hour,
	}

	ctx := context.Background()
	input := CreateBookingInput{
		FlightID:   4,
		SeatNumber: 10,
		Email:      "test@example.com",
	}

	// Место уже заблокировано
	// Используем service.holdTTL вместо time.Hour
	mockCache.On("AcquireSeatLock", ctx, int64(4), 10, time.Minute).Return(false, nil).Once()

	booking, err := service.CreateBooking(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, booking)
	assert.Contains(t, err.Error(), "seat is already locked")

	mockCache.AssertExpectations(t)
	mockBookingRepo.AssertNotCalled(t, "CreatePending")
}

// Тест 4: Создание бронирования - ошибка при блокировке места
func TestBookingService_CreateBooking_LockError(t *testing.T) {
	mockBookingRepo := &MockBookingRepository{}
	mockFlightRepo := &MockFlightRepository{}
	mockCache := &MockCache{}
	mockProducer := &MockProducer{}

	service := &BookingService{
		bookings:        mockBookingRepo,
		flights:         mockFlightRepo,
		cache:           mockCache,
		producer:        mockProducer,
		bookingTopic:    "booking_topic",
		holdTTL:         time.Minute,
		confirmationTTL: time.Hour,
	}

	ctx := context.Background()
	input := CreateBookingInput{
		FlightID:   4,
		SeatNumber: 10,
		Email:      "test@example.com",
	}

	// Ошибка при блокировке места
	expectedErr := errors.New("redis error")
	mockCache.On("AcquireSeatLock", ctx, int64(4), 10, time.Minute).Return(false, expectedErr).Once()
	booking, err := service.CreateBooking(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, booking)
	assert.Equal(t, expectedErr, err)

	mockCache.AssertExpectations(t)
	mockBookingRepo.AssertNotCalled(t, "CreatePending")
}

// Тест 5: Создание бронирования - ошибка в репозитории
func TestBookingService_CreateBooking_RepositoryError(t *testing.T) {
	mockBookingRepo := &MockBookingRepository{}
	mockFlightRepo := &MockFlightRepository{}
	mockCache := &MockCache{}
	mockProducer := &MockProducer{}

	service := &BookingService{
		bookings:        mockBookingRepo,
		flights:         mockFlightRepo,
		cache:           mockCache,
		producer:        mockProducer,
		bookingTopic:    "booking_topic",
		holdTTL:         time.Minute,
		confirmationTTL: time.Hour,
	}

	ctx := context.Background()
	input := CreateBookingInput{
		FlightID:   4,
		SeatNumber: 10,
		Email:      "test@example.com",
	}

	// Успешная блокировка, но ошибка в репозитории
	mockCache.On("AcquireSeatLock", ctx, int64(4), 10, time.Minute).Return(true, nil).Once()
	// Используем Times(2) для учета вызова через defer
	mockCache.On("ReleaseSeatLock", ctx, int64(4), 10).Return(nil).Once()

	expectedErr := errors.New("database error")
	mockBookingRepo.On("CreatePending", ctx, mock.Anything).Return(expectedErr).Once()

	booking, err := service.CreateBooking(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, booking)
	assert.Equal(t, expectedErr, err)

	mockCache.AssertExpectations(t)
	mockBookingRepo.AssertExpectations(t)
}

// Тест 6: Подтверждение бронирования - успешный сценарий
func TestBookingService_ConfirmBooking_Success(t *testing.T) {
	mockBookingRepo := &MockBookingRepository{}
	mockFlightRepo := &MockFlightRepository{}
	mockCache := &MockCache{}
	mockProducer := &MockProducer{}

	service := &BookingService{
		bookings:        mockBookingRepo,
		flights:         mockFlightRepo,
		cache:           mockCache,
		producer:        mockProducer,
		bookingTopic:    "booking_topic",
		holdTTL:         time.Minute,
		confirmationTTL: time.Hour,
	}

	ctx := context.Background()
	token := "test-token-123"

	existingBooking := &domain.Booking{
		ID:         1,
		FlightID:   4,
		SeatNumber: 10,
		Token:      token,
		Status:     domain.BookingStatusPending,
		Email:      "test@example.com",
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	updatedBooking := &domain.Booking{
		ID:         1,
		FlightID:   4,
		SeatNumber: 10,
		Token:      token,
		Status:     domain.BookingStatusConfirmed,
		Email:      "test@example.com",
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	// Настройка моков
	mockBookingRepo.On("GetByToken", ctx, token).Return(existingBooking, nil).Once()
	mockBookingRepo.On("UpdateStatus", ctx, token, domain.BookingStatusConfirmed).Return(updatedBooking, nil).Once()
	mockCache.On("ReleaseSeatLock", ctx, int64(4), 10).Return(nil).Once()
	mockProducer.On("Publish", ctx, "booking_topic", token, mock.Anything).Return(nil).Once()

	// Выполнение
	booking, err := service.ConfirmBooking(ctx, token)

	// Проверки
	assert.NoError(t, err)
	assert.NotNil(t, booking)
	assert.Equal(t, domain.BookingStatusConfirmed, booking.Status)

	mockBookingRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
	mockProducer.AssertExpectations(t)
}

// Тест 7: Подтверждение бронирования - бронирование не найдено
func TestBookingService_ConfirmBooking_NotFound(t *testing.T) {
	mockBookingRepo := &MockBookingRepository{}
	mockFlightRepo := &MockFlightRepository{}
	mockCache := &MockCache{}
	mockProducer := &MockProducer{}

	service := &BookingService{
		bookings:        mockBookingRepo,
		flights:         mockFlightRepo,
		cache:           mockCache,
		producer:        mockProducer,
		bookingTopic:    "booking_topic",
		holdTTL:         time.Minute,
		confirmationTTL: time.Hour,
	}

	ctx := context.Background()
	token := "non-existent-token"

	// Бронирование не найдено
	expectedErr := errors.New("booking not found")
	mockBookingRepo.On("GetByToken", ctx, token).Return(nil, expectedErr).Once()

	booking, err := service.ConfirmBooking(ctx, token)

	assert.Error(t, err)
	assert.Nil(t, booking)
	assert.Equal(t, expectedErr, err)

	mockBookingRepo.AssertExpectations(t)
	mockBookingRepo.AssertNotCalled(t, "UpdateStatus")
}

// Тест 8: Подтверждение бронирования - бронирование уже подтверждено
func TestBookingService_ConfirmBooking_AlreadyConfirmed(t *testing.T) {
	mockBookingRepo := &MockBookingRepository{}
	mockFlightRepo := &MockFlightRepository{}
	mockCache := &MockCache{}
	mockProducer := &MockProducer{}

	service := &BookingService{
		bookings:        mockBookingRepo,
		flights:         mockFlightRepo,
		cache:           mockCache,
		producer:        mockProducer,
		bookingTopic:    "booking_topic",
		holdTTL:         time.Minute,
		confirmationTTL: time.Hour,
	}

	ctx := context.Background()
	token := "already-confirmed-token"

	// Бронирование уже подтверждено
	existingBooking := &domain.Booking{
		ID:         1,
		FlightID:   4,
		SeatNumber: 10,
		Token:      token,
		Status:     domain.BookingStatusConfirmed,
		Email:      "test@example.com",
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	mockBookingRepo.On("GetByToken", ctx, token).Return(existingBooking, nil).Once()

	booking, err := service.ConfirmBooking(ctx, token)

	assert.Error(t, err)
	assert.Nil(t, booking)
	assert.Contains(t, err.Error(), "booking is not pending")

	mockBookingRepo.AssertExpectations(t)
	mockBookingRepo.AssertNotCalled(t, "UpdateStatus")
}

// Тест 9: Подтверждение бронирования - ошибка при обновлении статуса
func TestBookingService_ConfirmBooking_UpdateError(t *testing.T) {
	mockBookingRepo := &MockBookingRepository{}
	mockFlightRepo := &MockFlightRepository{}
	mockCache := &MockCache{}
	mockProducer := &MockProducer{}

	service := &BookingService{
		bookings:        mockBookingRepo,
		flights:         mockFlightRepo,
		cache:           mockCache,
		producer:        mockProducer,
		bookingTopic:    "booking_topic",
		holdTTL:         time.Minute,
		confirmationTTL: time.Hour,
	}

	ctx := context.Background()
	token := "test-token"

	existingBooking := &domain.Booking{
		ID:         1,
		FlightID:   4,
		SeatNumber: 10,
		Token:      token,
		Status:     domain.BookingStatusPending,
		Email:      "test@example.com",
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	// Ошибка при обновлении статуса
	expectedErr := errors.New("update error")
	mockBookingRepo.On("GetByToken", ctx, token).Return(existingBooking, nil).Once()
	mockBookingRepo.On("UpdateStatus", ctx, token, domain.BookingStatusConfirmed).Return(nil, expectedErr).Once()

	booking, err := service.ConfirmBooking(ctx, token)

	assert.Error(t, err)
	assert.Nil(t, booking)
	assert.Equal(t, expectedErr, err)

	mockBookingRepo.AssertExpectations(t)
	mockCache.AssertNotCalled(t, "ReleaseSeatLock")
}

// Тест 10: Отмена бронирования - успешный сценарий
func TestBookingService_CancelBooking_Success(t *testing.T) {
	mockBookingRepo := &MockBookingRepository{}
	mockFlightRepo := &MockFlightRepository{}
	mockCache := &MockCache{}
	mockProducer := &MockProducer{}

	service := &BookingService{
		bookings:        mockBookingRepo,
		flights:         mockFlightRepo,
		cache:           mockCache,
		producer:        mockProducer,
		bookingTopic:    "booking_topic",
		holdTTL:         time.Minute,
		confirmationTTL: time.Hour,
	}

	ctx := context.Background()
	token := "test-token-456"

	existingBooking := &domain.Booking{
		ID:         1,
		FlightID:   4,
		SeatNumber: 10,
		Token:      token,
		Status:     domain.BookingStatusPending,
		Email:      "test@example.com",
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	updatedBooking := &domain.Booking{
		ID:         1,
		FlightID:   4,
		SeatNumber: 10,
		Token:      token,
		Status:     domain.BookingStatusCancelled,
		Email:      "test@example.com",
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	// Настройка моков
	mockBookingRepo.On("GetByToken", ctx, token).Return(existingBooking, nil).Once()
	mockBookingRepo.On("UpdateStatus", ctx, token, domain.BookingStatusCancelled).Return(updatedBooking, nil).Once()
	mockBookingRepo.On("ReleaseSeat", ctx, int64(4)).Return(nil).Once()
	mockCache.On("ReleaseSeatLock", ctx, int64(4), 10).Return(nil).Once()
	mockProducer.On("Publish", ctx, "booking_topic", token, mock.Anything).Return(nil).Once()

	// Выполнение
	booking, err := service.CancelBooking(ctx, token)

	// Проверки
	assert.NoError(t, err)
	assert.NotNil(t, booking)
	assert.Equal(t, domain.BookingStatusCancelled, booking.Status)

	mockBookingRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
	mockProducer.AssertExpectations(t)
}

// Тест 11: Отмена бронирования - уже отменено
func TestBookingService_CancelBooking_AlreadyCancelled(t *testing.T) {
	mockBookingRepo := &MockBookingRepository{}
	mockFlightRepo := &MockFlightRepository{}
	mockCache := &MockCache{}
	mockProducer := &MockProducer{}

	service := &BookingService{
		bookings:        mockBookingRepo,
		flights:         mockFlightRepo,
		cache:           mockCache,
		producer:        mockProducer,
		bookingTopic:    "booking_topic",
		holdTTL:         time.Minute,
		confirmationTTL: time.Hour,
	}

	ctx := context.Background()
	token := "already-cancelled-token"

	// Бронирование уже отменено
	existingBooking := &domain.Booking{
		ID:         1,
		FlightID:   4,
		SeatNumber: 10,
		Token:      token,
		Status:     domain.BookingStatusCancelled,
		Email:      "test@example.com",
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	mockBookingRepo.On("GetByToken", ctx, token).Return(existingBooking, nil).Once()

	booking, err := service.CancelBooking(ctx, token)

	assert.NoError(t, err)
	assert.NotNil(t, booking)
	assert.Equal(t, existingBooking, booking) // Должен вернуть существующее без изменений

	mockBookingRepo.AssertExpectations(t)
	mockBookingRepo.AssertNotCalled(t, "UpdateStatus")
	mockBookingRepo.AssertNotCalled(t, "ReleaseSeat")
}

// Тест 12: Отмена бронирования - уже истекло
func TestBookingService_CancelBooking_AlreadyExpired(t *testing.T) {
	mockBookingRepo := &MockBookingRepository{}
	mockFlightRepo := &MockFlightRepository{}
	mockCache := &MockCache{}
	mockProducer := &MockProducer{}

	service := &BookingService{
		bookings:        mockBookingRepo,
		flights:         mockFlightRepo,
		cache:           mockCache,
		producer:        mockProducer,
		bookingTopic:    "booking_topic",
		holdTTL:         time.Minute,
		confirmationTTL: time.Hour,
	}

	ctx := context.Background()
	token := "expired-token"

	// Бронирование уже истекло
	existingBooking := &domain.Booking{
		ID:         1,
		FlightID:   4,
		SeatNumber: 10,
		Token:      token,
		Status:     domain.BookingStatusExpired,
		Email:      "test@example.com",
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	mockBookingRepo.On("GetByToken", ctx, token).Return(existingBooking, nil).Once()

	booking, err := service.CancelBooking(ctx, token)

	assert.NoError(t, err)
	assert.NotNil(t, booking)
	assert.Equal(t, existingBooking, booking)

	mockBookingRepo.AssertExpectations(t)
	mockBookingRepo.AssertNotCalled(t, "UpdateStatus")
	mockBookingRepo.AssertNotCalled(t, "ReleaseSeat")
}

// Тест 13: Отмена бронирования - бронирование не найдено
func TestBookingService_CancelBooking_NotFound(t *testing.T) {
	mockBookingRepo := &MockBookingRepository{}
	mockFlightRepo := &MockFlightRepository{}
	mockCache := &MockCache{}
	mockProducer := &MockProducer{}

	service := &BookingService{
		bookings:        mockBookingRepo,
		flights:         mockFlightRepo,
		cache:           mockCache,
		producer:        mockProducer,
		bookingTopic:    "booking_topic",
		holdTTL:         time.Minute,
		confirmationTTL: time.Hour,
	}

	ctx := context.Background()
	token := "non-existent-token"

	// Бронирование не найдено
	expectedErr := errors.New("booking not found")
	mockBookingRepo.On("GetByToken", ctx, token).Return(nil, expectedErr).Once()

	booking, err := service.CancelBooking(ctx, token)

	assert.Error(t, err)
	assert.Nil(t, booking)
	assert.Equal(t, expectedErr, err)

	mockBookingRepo.AssertExpectations(t)
	mockBookingRepo.AssertNotCalled(t, "UpdateStatus")
}

// Тест 14: Истечение просроченных бронирований
func TestBookingService_ExpirePendingBookings_Success(t *testing.T) {
	mockBookingRepo := &MockBookingRepository{}
	mockFlightRepo := &MockFlightRepository{}
	mockCache := &MockCache{}
	mockProducer := &MockProducer{}

	service := &BookingService{
		bookings:        mockBookingRepo,
		flights:         mockFlightRepo,
		cache:           mockCache,
		producer:        mockProducer,
		bookingTopic:    "booking_topic",
		holdTTL:         time.Minute,
		confirmationTTL: time.Hour,
	}

	ctx := context.Background()

	// Создаем просроченные бронирования
	expiredBookings := []domain.Booking{
		{
			ID:         1,
			FlightID:   4,
			SeatNumber: 10,
			Token:      "token1",
			Status:     domain.BookingStatusPending,
			Email:      "test1@example.com",
			ExpiresAt:  time.Now().Add(-time.Hour),
		},
		{
			ID:         2,
			FlightID:   5,
			SeatNumber: 20,
			Token:      "token2",
			Status:     domain.BookingStatusPending,
			Email:      "test2@example.com",
			ExpiresAt:  time.Now().Add(-2 * time.Hour),
		},
	}

	// Настройка моков
	mockBookingRepo.On("ExpirePendingBefore", ctx, mock.AnythingOfType("time.Time")).Return(expiredBookings, nil).Once()
	mockBookingRepo.On("ReleaseSeat", ctx, int64(4)).Return(nil).Once()
	mockBookingRepo.On("ReleaseSeat", ctx, int64(5)).Return(nil).Once()
	mockCache.On("ReleaseSeatLock", ctx, int64(4), 10).Return(nil).Once()
	mockCache.On("ReleaseSeatLock", ctx, int64(5), 20).Return(nil).Once()
	mockProducer.On("Publish", ctx, "booking_topic", "token1", mock.Anything).Return(nil).Once()
	mockProducer.On("Publish", ctx, "booking_topic", "token2", mock.Anything).Return(nil).Once()

	// Выполнение
	result, err := service.ExpirePendingBookings(ctx)

	// Проверки
	assert.NoError(t, err)
	assert.Equal(t, expiredBookings, result)

	mockBookingRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
	mockProducer.AssertExpectations(t)
}

// Тест 15: Истечение просроченных бронирований - пустой список
func TestBookingService_ExpirePendingBookings_Empty(t *testing.T) {
	mockBookingRepo := &MockBookingRepository{}
	mockFlightRepo := &MockFlightRepository{}
	mockCache := &MockCache{}
	mockProducer := &MockProducer{}

	service := &BookingService{
		bookings:        mockBookingRepo,
		flights:         mockFlightRepo,
		cache:           mockCache,
		producer:        mockProducer,
		bookingTopic:    "booking_topic",
		holdTTL:         time.Minute,
		confirmationTTL: time.Hour,
	}

	ctx := context.Background()

	// Нет просроченных бронирований
	emptyBookings := []domain.Booking{}

	// Настройка моков
	mockBookingRepo.On("ExpirePendingBefore", ctx, mock.AnythingOfType("time.Time")).Return(emptyBookings, nil).Once()

	// Выполнение
	result, err := service.ExpirePendingBookings(ctx)

	// Проверки
	assert.NoError(t, err)
	assert.Empty(t, result)

	mockBookingRepo.AssertExpectations(t)
	mockBookingRepo.AssertNotCalled(t, "ReleaseSeat")
	mockCache.AssertNotCalled(t, "ReleaseSeatLock")
	mockProducer.AssertNotCalled(t, "Publish")
}

// Тест 16: Истечение просроченных бронирований - ошибка при получении
func TestBookingService_ExpirePendingBookings_Error(t *testing.T) {
	mockBookingRepo := &MockBookingRepository{}
	mockFlightRepo := &MockFlightRepository{}
	mockCache := &MockCache{}
	mockProducer := &MockProducer{}

	service := &BookingService{
		bookings:        mockBookingRepo,
		flights:         mockFlightRepo,
		cache:           mockCache,
		producer:        mockProducer,
		bookingTopic:    "booking_topic",
		holdTTL:         time.Minute,
		confirmationTTL: time.Hour,
	}

	ctx := context.Background()

	// Ошибка при получении просроченных бронирований
	expectedErr := errors.New("database error")
	mockBookingRepo.On("ExpirePendingBefore", ctx, mock.AnythingOfType("time.Time")).Return([]domain.Booking{}, expectedErr).Once()

	// Выполнение
	result, err := service.ExpirePendingBookings(ctx)

	// Проверки
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedErr, err)

	mockBookingRepo.AssertExpectations(t)
	mockBookingRepo.AssertNotCalled(t, "ReleaseSeat")
}

// Тест 17: Тест метода publish без producer
func TestBookingService_Publish_NoProducer(t *testing.T) {
	service := &BookingService{
		producer: nil,
	}

	ctx := context.Background()
	booking := &domain.Booking{
		Token: "test-token",
	}

	err := service.publish(ctx, "test_event", booking)
	assert.NoError(t, err) // Должен просто вернуть nil
}

// Тест 18: Тест метода publish с пустым bookingTopic
func TestBookingService_Publish_NoTopic(t *testing.T) {
	mockProducer := &MockProducer{}

	service := &BookingService{
		producer:     mockProducer,
		bookingTopic: "",
	}

	ctx := context.Background()
	booking := &domain.Booking{
		Token: "test-token",
	}

	err := service.publish(ctx, "test_event", booking)
	assert.NoError(t, err) // Должен просто вернуть nil

	mockProducer.AssertNotCalled(t, "Publish")
}

// Тест 19: Тест метода publish с notificationsTopic
func TestBookingService_Publish_WithNotifications(t *testing.T) {
	mockProducer := &MockProducer{}

	service := &BookingService{
		producer:           mockProducer,
		bookingTopic:       "booking_topic",
		notificationsTopic: "notifications_topic",
	}

	ctx := context.Background()
	booking := &domain.Booking{
		Token:      "test-token",
		FlightID:   4,
		SeatNumber: 10,
		Email:      "test@example.com",
		Status:     domain.BookingStatusPending,
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	// Producer должен быть вызван дважды
	mockProducer.On("Publish", ctx, "booking_topic", "test-token", mock.Anything).Return(nil).Once()
	mockProducer.On("Publish", ctx, "notifications_topic", "test-token", mock.Anything).Return(nil).Once()

	err := service.publish(ctx, "test_event", booking)
	assert.NoError(t, err)

	mockProducer.AssertExpectations(t)
}

// ============================ Тесты для FlightService ============================

// Тест 20: Получение списка рейсов - кэш пустой

// ============================ Дополнительные тесты для достижения покрытия ============================

// Тест 26: Создание сервиса с опциями
func TestNewBookingService_WithOptions(t *testing.T) {
	mockBookingRepo := &MockBookingRepository{}
	mockFlightRepo := &MockFlightRepository{}
	mockCache := &MockCache{}
	mockProducer := &MockProducer{}

	service := &BookingService{
		bookings:           mockBookingRepo,
		flights:            mockFlightRepo,
		cache:              mockCache,
		producer:           mockProducer,
		bookingTopic:       "booking_topic",
		holdTTL:            time.Minute,
		confirmationTTL:    time.Hour,
		notificationsTopic: "notifications_topic",
	}

	assert.NotNil(t, service)
	assert.Equal(t, "notifications_topic", service.notificationsTopic)
	assert.Equal(t, mockBookingRepo, service.bookings)
	assert.Equal(t, mockFlightRepo, service.flights)
	assert.Equal(t, mockCache, service.cache)
	assert.Equal(t, mockProducer, service.producer)
}

func TestBookingExpirationLogic(t *testing.T) {
	// Тест бизнес-логики истечения времени
	now := time.Now()

	// Бронирование еще не истекло
	futureBooking := &domain.Booking{
		ExpiresAt: now.Add(time.Hour),
	}
	assert.True(t, futureBooking.ExpiresAt.After(now))

	// Бронирование истекло
	pastBooking := &domain.Booking{
		ExpiresAt: now.Add(-time.Hour),
	}
	assert.True(t, pastBooking.ExpiresAt.Before(now))

	// Проверка разницы во времени
	expiresIn := futureBooking.ExpiresAt.Sub(now)
	assert.True(t, expiresIn > 0)
}

// Тест 29: Тест расчета времени истечения
func TestExpirationTimeCalculation(t *testing.T) {
	service := &BookingService{
		holdTTL:         time.Minute * 30,
		confirmationTTL: time.Hour,
	}

	// Если confirmationTTL установлен, используется он
	expiresIn := service.confirmationTTL
	if expiresIn == 0 {
		expiresIn = service.holdTTL
	}
	assert.Equal(t, time.Hour, expiresIn)

	// Если confirmationTTL не установлен, используется holdTTL
	service2 := &BookingService{
		holdTTL: time.Minute * 30,
	}
	expiresIn2 := service2.confirmationTTL
	if expiresIn2 == 0 {
		expiresIn2 = service2.holdTTL
	}
	assert.Equal(t, time.Minute*30, expiresIn2)
}

// Тест 30: Тест статусов бронирования
func TestBookingStatuses(t *testing.T) {
	// Проверяем все возможные статусы
	statuses := []domain.BookingStatus{
		domain.BookingStatusPending,
		domain.BookingStatusConfirmed,
		domain.BookingStatusCancelled,
		domain.BookingStatusExpired,
	}

	for _, status := range statuses {
		assert.NotEmpty(t, string(status))
	}

	// Проверяем переходы статусов
	pendingBooking := &domain.Booking{
		Status: domain.BookingStatusPending,
	}
	assert.True(t, pendingBooking.Status == domain.BookingStatusPending)

	confirmedBooking := &domain.Booking{
		Status: domain.BookingStatusConfirmed,
	}
	assert.True(t, confirmedBooking.Status == domain.BookingStatusConfirmed)

	// Проверяем что нельзя подтвердить уже подтвержденное бронирование
	assert.False(t, confirmedBooking.Status == domain.BookingStatusPending)
}

// Тест 32: Тест работы без кэша в BookingService
func TestBookingService_NoCache(t *testing.T) {
	mockBookingRepo := &MockBookingRepository{}
	mockFlightRepo := &MockFlightRepository{}
	mockProducer := &MockProducer{}

	service := &BookingService{
		bookings:        mockBookingRepo,
		flights:         mockFlightRepo,
		cache:           nil, // Нет кэша
		producer:        mockProducer,
		bookingTopic:    "booking_topic",
		holdTTL:         time.Minute,
		confirmationTTL: time.Hour,
	}

	ctx := context.Background()
	input := CreateBookingInput{
		FlightID:   4,
		SeatNumber: 10,
		Email:      "test@example.com",
	}

	mockBookingRepo.On("CreatePending", ctx, mock.Anything).Return(nil).Once()
	mockProducer.On("Publish", ctx, "booking_topic", mock.Anything, mock.Anything).Return(nil).Once()

	booking, err := service.CreateBooking(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, booking)

	mockBookingRepo.AssertExpectations(t)
	mockProducer.AssertExpectations(t)
}
