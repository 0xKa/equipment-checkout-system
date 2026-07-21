package handlers

import (
	"errors"
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

func writeJSONError(c *echo.Context, err error) error {
	if echo.StatusCode(err) == http.StatusRequestEntityTooLarge {
		return err
	}

	return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, err.Error())
}

func writeServiceError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, types.ErrInvalidInput):
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "asset_tag and name must not be empty")
	case errors.Is(err, types.ErrInvalidCategoryID):
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "category_id must be a positive integer")
	case errors.Is(err, types.ErrInvalidCategoryInput):
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "category name must not be empty")
	case errors.Is(err, types.ErrItemNotFound):
		return writeAPIError(c, http.StatusNotFound, types.ErrorCodeItemNotFound, "item not found")
	case errors.Is(err, types.ErrAssetTagConflict):
		return writeAPIError(c, http.StatusConflict, types.ErrorCodeAssetTagConflict, "asset_tag already exists")
	case errors.Is(err, types.ErrSerialNumberConflict):
		return writeAPIError(c, http.StatusConflict, types.ErrorCodeSerialNumberConflict, "serial_number already exists")
	case errors.Is(err, types.ErrCategoryNotFound):
		return writeAPIError(c, http.StatusNotFound, types.ErrorCodeCategoryNotFound, "category not found")
	case errors.Is(err, types.ErrCategoryNameConflict):
		return writeAPIError(c, http.StatusConflict, types.ErrorCodeCategoryNameConflict, "category name already exists")
	case errors.Is(err, types.ErrCategoryInUse):
		return writeAPIError(c, http.StatusConflict, types.ErrorCodeCategoryInUse, "category is assigned to one or more items")
	case errors.Is(err, types.ErrItemInUse):
		return writeAPIError(c, http.StatusConflict, types.ErrorCodeItemInUse, "item is referenced by existing records")
	default:
		return err
	}
}

func requestID(c *echo.Context) string {
	requestID := c.Request().Header.Get(echo.HeaderXRequestID)
	if requestID == "" {
		requestID = c.Response().Header().Get(echo.HeaderXRequestID)
	}
	return requestID
}
