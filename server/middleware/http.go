package middleware

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v5"
	echomiddleware "github.com/labstack/echo/v5/middleware"
)

const requestTimeout = 15 * time.Second

// Register installs process-wide safeguards in correlation and logging order.
func Register(server *echo.Echo) {
	server.Use(
		echomiddleware.RequestID(),
		requestLogger(),
		echomiddleware.ContextTimeout(requestTimeout),
		echomiddleware.Recover(),
		echomiddleware.SecureWithConfig(echomiddleware.SecureConfig{
			XSSProtection:         "0",
			ContentTypeNosniff:    "nosniff",
			XFrameOptions:         "DENY",
			ContentSecurityPolicy: "default-src 'none'; " + "frame-ancestors 'none'",
			ReferrerPolicy:        "no-referrer",
		}),
	)
}

func requestLogger() echo.MiddlewareFunc {
	return echomiddleware.RequestLoggerWithConfig(echomiddleware.RequestLoggerConfig{
		HandleError:      true,
		LogLatency:       true,
		LogRemoteIP:      true,
		LogMethod:        true,
		LogRoutePath:     true,
		LogRequestID:     true,
		LogStatus:        true,
		LogContentLength: true,
		LogResponseSize:  true,
		LogValuesFunc: func(c *echo.Context, values echomiddleware.RequestLoggerValues) error {
			level := slog.LevelInfo
			switch {
			case values.Status >= 500:
				level = slog.LevelError
			case values.Status >= 400:
				level = slog.LevelWarn
			}

			attributes := []slog.Attr{
				slog.String("request_id", values.RequestID),
				slog.String("method", values.Method),
				slog.String("route", values.RoutePath),
				slog.Int("status", values.Status),
				slog.Duration("latency", values.Latency),
				slog.String("remote_ip", values.RemoteIP),
				slog.String("bytes_in", values.ContentLength),
				slog.Int64("bytes_out", values.ResponseSize),
			}
			if values.Error != nil {
				attributes = append(attributes, slog.String("error", values.Error.Error()))
			}

			// Echo writes through the Zap-backed slog adapter configured by the serve command.
			c.Logger().LogAttrs(c.Request().Context(), level, "HTTP request", attributes...)
			return nil
		},
	})
}
