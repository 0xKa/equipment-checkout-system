package routes

import (
	"github.com/0xKa/equipment-checkout-system/server/handlers"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const maxRequestBodyBytes int64 = 1024 * 1024

func Register(
	e *echo.Echo,
	healthHandler *handlers.Health,
	itemHandler *handlers.Items,
	categoryHandler *handlers.Categories,
) {
	e.GET("/health", healthHandler.Get)

	v1 := e.Group("/api/v1")

	items := v1.Group("/items", middleware.BodyLimit(maxRequestBodyBytes))
	items.POST("", itemHandler.Create)
	items.GET("", itemHandler.List)
	items.GET("/:id", itemHandler.Get)
	items.PUT("/:id", itemHandler.Update)
	items.DELETE("/:id", itemHandler.Delete)

	categories := v1.Group("/categories", middleware.BodyLimit(maxRequestBodyBytes))
	categories.POST("", categoryHandler.Create)
	categories.GET("", categoryHandler.List)
	categories.GET("/:id", categoryHandler.Get)
	categories.PUT("/:id", categoryHandler.Update)
	categories.DELETE("/:id", categoryHandler.Delete)
}
