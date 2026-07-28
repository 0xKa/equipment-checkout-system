package routes

import (
	"github.com/0xKa/equipment-checkout-system/server/handlers"
	appmiddleware "github.com/0xKa/equipment-checkout-system/server/middleware"
	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/labstack/echo/v5"
	echomiddleware "github.com/labstack/echo/v5/middleware"
)

const maxRequestBodyBytes int64 = 1024 * 1024

func Register(
	e *echo.Echo,
	healthHandler *handlers.Health,
	itemHandler *handlers.Items,
	categoryHandler *handlers.Categories,
	userHandler *handlers.Users,
	checkoutHandler *handlers.Checkouts,
	requireBearer echo.MiddlewareFunc,
) {
	e.GET("/health", healthHandler.Get)
	e.GET("/ready", healthHandler.Ready)

	v1 := e.Group("/api/v1", requireBearer)

	requireInventoryRead := appmiddleware.RequireCapability(
		types.CapabilityInventoryRead,
	)
	requireInventoryManage := appmiddleware.RequireCapability(
		types.CapabilityInventoryManage,
	)
	requireUsersManage := appmiddleware.RequireCapability(
		types.CapabilityUsersManage,
	)
	requireCheckoutManage := appmiddleware.RequireCapability(
		types.CapabilityCheckoutManage,
	)
	requireCheckoutHistoryReadAll := appmiddleware.RequireCapability(
		types.CapabilityCheckoutHistoryReadAll,
	)

	items := v1.Group("/items", echomiddleware.BodyLimit(maxRequestBodyBytes))
	items.POST("", itemHandler.Create, requireInventoryManage)
	items.GET("", itemHandler.List, requireInventoryRead)
	items.GET("/:id", itemHandler.Get, requireInventoryRead)
	items.PUT("/:id", itemHandler.Update, requireInventoryManage)
	items.DELETE("/:id", itemHandler.Delete, requireInventoryManage)
	items.POST("/:id/checkouts", checkoutHandler.CreateForItem, requireCheckoutManage)
	items.GET("/:id/checkouts", checkoutHandler.ListForItem, requireCheckoutHistoryReadAll)

	categories := v1.Group("/categories", echomiddleware.BodyLimit(maxRequestBodyBytes))
	categories.POST("", categoryHandler.Create, requireInventoryManage)
	categories.GET("", categoryHandler.List, requireInventoryRead)
	categories.GET("/:id", categoryHandler.Get, requireInventoryRead)
	categories.PUT("/:id", categoryHandler.Update, requireInventoryManage)
	categories.DELETE("/:id", categoryHandler.Delete, requireInventoryManage)

	users := v1.Group("/users", echomiddleware.BodyLimit(maxRequestBodyBytes))
	users.POST("", userHandler.Create, requireUsersManage)
	users.GET("", userHandler.List, requireUsersManage)
	users.GET("/:id", userHandler.Get, requireUsersManage)
	users.PUT("/:id", userHandler.Update, requireUsersManage)
	users.PATCH("/:id/status", userHandler.SetStatus, requireUsersManage)

	checkouts := v1.Group("/checkouts")
	checkouts.GET("", checkoutHandler.List, requireCheckoutHistoryReadAll)
	checkouts.GET("/:id", checkoutHandler.Get, requireCheckoutHistoryReadAll)
	checkouts.POST("/:id/return", checkoutHandler.Return, requireCheckoutManage)

	v1.GET("/me", userHandler.Me)
}
