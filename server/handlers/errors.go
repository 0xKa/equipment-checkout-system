package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/labstack/echo/v5"
)

// Writes errors returned through Echo using the public API envelope.
func HTTPErrorHandler(c *echo.Context, err error) {
	response, _ := echo.UnwrapResponse(c.Response())
	if response != nil && response.Committed {
		return
	}

	if errors.Is(err, context.DeadlineExceeded) {
		if writeErr := writeAPIError(
			c,
			http.StatusGatewayTimeout,
			types.ErrorCodeRequestTimeout,
			"request timed out",
		); writeErr != nil {
			c.Logger().Error("write HTTP error response", "error", writeErr)
		}
		return
	}

	if status, code, message, ok := serviceError(err); ok {
		if writeErr := writeAPIError(c, status, code, message); writeErr != nil {
			c.Logger().Error("write HTTP error response", "error", writeErr)
		}
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

func writeJSONError(c *echo.Context, err error) error {
	if echo.StatusCode(err) == http.StatusRequestEntityTooLarge {
		return err
	}

	return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, err.Error())
}

func writeServiceError(c *echo.Context, err error) error {
	status, code, message, ok := serviceError(err)
	if !ok {
		return err
	}

	return writeAPIError(c, status, code, message)
}

// Maps known domain failures to public HTTP errors.
func serviceError(err error) (int, string, string, bool) {
	switch {
	case errors.Is(err, types.ErrInvalidInput):
		return http.StatusBadRequest, types.ErrorCodeInvalidRequest, "asset_tag and name must not be empty", true
	case errors.Is(err, types.ErrInvalidCategoryID):
		return http.StatusBadRequest, types.ErrorCodeInvalidRequest, "category_id must be a positive integer", true
	case errors.Is(err, types.ErrInvalidCategoryInput):
		return http.StatusBadRequest, types.ErrorCodeInvalidRequest, "category name must not be empty", true
	case errors.Is(err, types.ErrItemNotFound):
		return http.StatusNotFound, types.ErrorCodeItemNotFound, "item not found", true
	case errors.Is(err, types.ErrAssetTagConflict):
		return http.StatusConflict, types.ErrorCodeAssetTagConflict, "asset_tag already exists", true
	case errors.Is(err, types.ErrSerialNumberConflict):
		return http.StatusConflict, types.ErrorCodeSerialNumberConflict, "serial_number already exists", true
	case errors.Is(err, types.ErrCategoryNotFound):
		return http.StatusNotFound, types.ErrorCodeCategoryNotFound, "category not found", true
	case errors.Is(err, types.ErrCategoryNameConflict):
		return http.StatusConflict, types.ErrorCodeCategoryNameConflict, "category name already exists", true
	case errors.Is(err, types.ErrCategoryInUse):
		return http.StatusConflict, types.ErrorCodeCategoryInUse, "category is assigned to one or more items", true
	case errors.Is(err, types.ErrItemInUse):
		return http.StatusConflict, types.ErrorCodeItemInUse, "item is referenced by existing records", true
	case errors.Is(err, types.ErrInvalidUserID):
		return http.StatusBadRequest, types.ErrorCodeInvalidRequest, "user id must be a positive integer", true
	case errors.Is(err, types.ErrInvalidUserInput):
		return http.StatusBadRequest, types.ErrorCodeInvalidRequest, "username and display_name are required; email must be valid when provided", true
	case errors.Is(err, types.ErrUserNotFound):
		return http.StatusNotFound, types.ErrorCodeUserNotFound, "user not found", true
	case errors.Is(err, types.ErrUsernameConflict):
		return http.StatusConflict, types.ErrorCodeUsernameConflict, "username already exists", true
	case errors.Is(err, types.ErrUserEmailConflict):
		return http.StatusConflict, types.ErrorCodeUserEmailConflict, "email already exists", true
	case errors.Is(err, types.ErrActorRequired):
		return http.StatusBadRequest, types.ErrorCodeInvalidActor, "X-Actor-User-ID header is required", true
	case errors.Is(err, types.ErrInvalidActor):
		return http.StatusBadRequest, types.ErrorCodeInvalidActor, "X-Actor-User-ID must contain one positive user id", true
	case errors.Is(err, types.ErrActorInactive):
		return http.StatusConflict, types.ErrorCodeActorInactive, "actor user is inactive", true
	default:
		return 0, "", "", false
	}
}

func requestID(c *echo.Context) string {
	requestID := c.Request().Header.Get(echo.HeaderXRequestID)
	if requestID == "" {
		requestID = c.Response().Header().Get(echo.HeaderXRequestID)
	}
	return requestID
}
