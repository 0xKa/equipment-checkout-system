package middleware

import (
	"time"

	"github.com/labstack/echo/v5"
	echomiddleware "github.com/labstack/echo/v5/middleware"
	"go.uber.org/zap"
)

const requestTimeout = 15 * time.Second

// Register installs process-wide safeguards in correlation and logging order.
func Register(server *echo.Echo, log *zap.Logger) {
	server.Use(
		echomiddleware.RequestID(),
		requestLogger(log),
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

func requestLogger(log *zap.Logger) echo.MiddlewareFunc {
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
		LogValuesFunc: func(_ *echo.Context, values echomiddleware.RequestLoggerValues) error {
			fields := []zap.Field{
				zap.String("request_id", values.RequestID),
				zap.String("method", values.Method),
				zap.String("route", values.RoutePath),
				zap.Int("status", values.Status),
				zap.Duration("latency", values.Latency),
				zap.String("remote_ip", values.RemoteIP),
				zap.String("bytes_in", values.ContentLength),
				zap.Int64("bytes_out", values.ResponseSize),
			}
			if values.Error != nil {
				fields = append(fields, zap.Error(values.Error))
			}

			switch {
			case values.Status >= 500:
				log.Error("HTTP request", fields...)
			case values.Status >= 400:
				log.Warn("HTTP request", fields...)
			default:
				log.Info("HTTP request", fields...)
			}
			return nil
		},
	})
}
