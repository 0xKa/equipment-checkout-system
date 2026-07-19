package routes

import (
	"github.com/0xKa/equipment-checkout-system/server/handlers"
	"github.com/labstack/echo/v5"
)

func Register(e *echo.Echo, healthHandler *handlers.Health) {
	e.GET("/health", healthHandler.Get)
}
