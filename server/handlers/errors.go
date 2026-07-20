package handlers

import (
	"net/http"

	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/labstack/echo/v5"
)

// converts errors returned through Echo into the public API envelope.
func HTTPErrorHandler(c *echo.Context, err error) {
	response, _ := echo.UnwrapResponse(c.Response())
	if response != nil && response.Committed {
		return
	}

	status := echo.StatusCode(err)
	code := types.ErrorCodeInvalidRequest
	message := http.StatusText(status)

	switch {
	case status == http.StatusRequestEntityTooLarge:
		message = "request body is too large"
	case status == http.StatusNotFound:
		message = "route not found"
	case status == http.StatusMethodNotAllowed:
		message = "method not allowed"
	case status >= http.StatusBadRequest && status < http.StatusInternalServerError:
		if message == "" {
			message = "invalid request"
		}
	default:
		// Unexpected and recovered errors must never expose their internal cause.
		status = http.StatusInternalServerError
		code = types.ErrorCodeInternal
		message = "internal server error"
	}

	if c.Request().Method == http.MethodHead {
		if writeErr := c.NoContent(status); writeErr != nil {
			c.Logger().Error("write HTTP error response", "error", writeErr)
		}
		return
	}

	if writeErr := writeAPIError(c, status, code, message); writeErr != nil {
		c.Logger().Error("write HTTP error response", "error", writeErr)
	}
}

func writeAPIError(c *echo.Context, status int, code, message string) error {
	return c.JSON(status, types.ErrorResponse{
		Error: types.ErrorDetail{
			Code:      code,
			Message:   message,
			RequestID: requestID(c),
		},
	})
}

func requestID(c *echo.Context) string {
	requestID := c.Request().Header.Get(echo.HeaderXRequestID)
	if requestID == "" {
		requestID = c.Response().Header().Get(echo.HeaderXRequestID)
	}
	return requestID
}
