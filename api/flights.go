package api

import (
	"net/http"
	"strconv"

	"github.com/Domenick1991/airbooking/internal/service/flights"
	"github.com/gin-gonic/gin"
)

type FlightHandler struct {
	service flights.FlightUseCase
}

func NewFlightHandler(service flights.FlightUseCase) *FlightHandler {
	return &FlightHandler{service: service}
}

func (h *FlightHandler) Register(router *gin.RouterGroup) {
	router.GET("/", h.list)
	router.GET("/:id", h.get)
}

func (h *FlightHandler) list(c *gin.Context) {
	flights, err := h.service.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, flights)
}

func (h *FlightHandler) get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	flight, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, flight)
}
