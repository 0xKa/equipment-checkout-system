package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/labstack/echo/v5"
)

const readinessTimeout = 2 * time.Second

type databasePinger interface {
	Ping(context.Context) error
}

type Health struct {
	database databasePinger
}

func NewHealth(database databasePinger) *Health {
	return &Health{database: database}
}

func (h *Health) Get(c *echo.Context) error {
	return c.String(http.StatusOK, "healthy")
}

func (h *Health) Ready(c *echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), readinessTimeout)
	defer cancel()

	if err := h.database.Ping(ctx); err != nil {
		return writeAPIError(
			c,
			http.StatusServiceUnavailable,
			types.ErrorCodeServiceUnavailable,
			"service is not ready",
		)
	}

	return c.String(http.StatusOK, "ready")
}
