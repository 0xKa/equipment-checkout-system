package handlers

import (
	"net/http"

	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/labstack/echo/v5"
)

type Health struct{}

func NewHealth() *Health {
	return &Health{}
}

func (h *Health) Get(c *echo.Context) error {
	return c.JSON(http.StatusOK, types.HealthResponse{Status: "healthy"})
}
