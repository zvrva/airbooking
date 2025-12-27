package api

import (
	"net/http"
	"time"

	"github.com/Domenick1991/airbooking/internal/service/booking"
	"github.com/gin-gonic/gin"
)

type BookingHandler struct {
	service booking.BookingUseCase
}

type createBookingRequest struct {
	FlightID   int64  `json:"flight_id"`
	SeatNumber int    `json:"seat_number"`
	Email      string `json:"email"`
}

type bookingResponse struct {
	Token      string `json:"token"`
	Status     string `json:"status"`
	ExpiresAt  string `json:"expires_at"`
	FlightID   int64  `json:"flight_id"`
	SeatNumber int    `json:"seat_number"`
	Email      string `json:"email"`
}

func NewBookingHandler(service booking.BookingUseCase) *BookingHandler {
	return &BookingHandler{service: service}
}

func (h *BookingHandler) Register(router *gin.RouterGroup) {
	router.POST("/", h.create)
	router.PUT("/:token", h.confirm)
	router.DELETE("/:token", h.cancel)
}

func (h *BookingHandler) create(c *gin.Context) {
	var req createBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	booking, err := h.service.CreateBooking(c.Request.Context(), booking.CreateBookingInput{
		FlightID:   req.FlightID,
		SeatNumber: req.SeatNumber,
		Email:      req.Email,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, bookingResponse{
		Token:      booking.Token,
		Status:     string(booking.Status),
		ExpiresAt:  booking.ExpiresAt.Format(time.RFC3339),
		FlightID:   booking.FlightID,
		SeatNumber: booking.SeatNumber,
		Email:      booking.Email,
	})
}

func (h *BookingHandler) confirm(c *gin.Context) {
	token := c.Param("token")
	booking, err := h.service.ConfirmBooking(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, bookingResponse{
		Token:      booking.Token,
		Status:     string(booking.Status),
		ExpiresAt:  booking.ExpiresAt.Format(time.RFC3339),
		FlightID:   booking.FlightID,
		SeatNumber: booking.SeatNumber,
		Email:      booking.Email,
	})
}

func (h *BookingHandler) cancel(c *gin.Context) {
	token := c.Param("token")
	booking, err := h.service.CancelBooking(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, bookingResponse{
		Token:      booking.Token,
		Status:     string(booking.Status),
		ExpiresAt:  booking.ExpiresAt.Format(time.RFC3339),
		FlightID:   booking.FlightID,
		SeatNumber: booking.SeatNumber,
		Email:      booking.Email,
	})
}
