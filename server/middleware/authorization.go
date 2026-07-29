package middleware

import (
	"slices"

	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/labstack/echo/v5"
)

// RequireCapability authorizes the authenticated actor for a coarse
// application operation. Resource ownership remains a service concern.
func RequireCapability(capability types.Capability) echo.MiddlewareFunc {
	return RequireAnyCapability(capability)
}

// RequireAnyCapability authorizes an actor that holds at least one requested
// application capability.
func RequireAnyCapability(capabilities ...types.Capability) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			actor, ok := types.ActorFromContext(c.Request().Context())
			if !ok {
				return types.ErrAuthenticationRequired
			}

			if slices.ContainsFunc(capabilities, actor.Capabilities.Has) {
				return next(c)
			}

			return types.ErrForbidden
		}
	}
}
