package middleware

import (
	"strings"

	"github.com/0xKa/equipment-checkout-system/server/services"
	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/0xKa/equipment-checkout-system/server/utils"
	"github.com/labstack/echo/v5"
)

const HeaderActorUserID = "X-Actor-User-ID"

// Resolves the provisional actor and stores it in request context.
func RequireActor(resolver services.ActorResolver) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			values := c.Request().Header.Values(HeaderActorUserID)
			if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
				return types.ErrActorRequired
			}
			if len(values) != 1 {
				return types.ErrInvalidActor
			}

			userID, err := utils.ParseID(strings.TrimSpace(values[0]))
			if err != nil {
				return types.ErrInvalidActor
			}

			actor, err := resolver.Resolve(c.Request().Context(), userID)
			if err != nil {
				return err
			}

			ctx := types.ContextWithActor(c.Request().Context(), actor)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}
