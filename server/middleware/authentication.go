package middleware

import (
	"net/http"
	"strings"

	"github.com/0xKa/equipment-checkout-system/server/services"
	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/labstack/echo/v5"
)

// RequireBearer authenticates exactly one Bearer credential and stores its
// linked local actor in request context.
func RequireBearer(authentication services.AuthenticationService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			rawToken, err := bearerCredential(c.Request().Header)
			if err != nil {
				return err
			}

			actor, err := authentication.Authenticate(c.Request().Context(), rawToken)
			if err != nil {
				return err
			}

			ctx := types.ContextWithActor(c.Request().Context(), actor)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

func bearerCredential(header http.Header) (string, error) {
	values := header.Values(echo.HeaderAuthorization)
	if len(values) != 1 {
		return "", types.ErrAuthenticationRequired
	}

	value := strings.TrimSpace(values[0])
	if value == "" || strings.Contains(value, ",") {
		return "", types.ErrAuthenticationRequired
	}

	fields := strings.Fields(value)
	if len(fields) != 2 ||
		!strings.EqualFold(fields[0], "Bearer") ||
		fields[1] == "" {
		return "", types.ErrAuthenticationRequired
	}

	return fields[1], nil
}
