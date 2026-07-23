package handlers

import (
	"net/http"
	"strconv"

	"github.com/0xKa/equipment-checkout-system/server/services"
	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/0xKa/equipment-checkout-system/server/utils"
	"github.com/labstack/echo/v5"
)

const (
	checkoutQueryLimit  = "limit"
	checkoutQueryOffset = "offset"

	defaultCheckoutPageLimit int32 = 50
	maxCheckoutPageLimit     int32 = 100
)

type Checkouts struct {
	service services.CheckoutService
}

func NewCheckouts(service services.CheckoutService) *Checkouts {
	return &Checkouts{service: service}
}

func (h *Checkouts) CreateForItem(c *echo.Context) error {
	itemID, err := utils.ParseID(c.Param("id"))
	if err != nil {
		return writeAPIError(
			c,
			http.StatusBadRequest,
			types.ErrorCodeInvalidRequest,
			"item id must be a positive integer",
		)
	}

	var input types.CreateCheckoutRequest
	if err := utils.DecodeJSON(c, &input); err != nil {
		return writeJSONError(c, err)
	}

	checkout, err := h.service.Checkout(
		c.Request().Context(),
		itemID,
		input,
		types.WorkflowMetadata{RequestID: requestID(c)},
	)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusCreated, types.CheckoutResponse{Data: checkout})
}

func (h *Checkouts) Return(c *echo.Context) error {
	checkoutID, err := utils.ParseID(c.Param("id"))
	if err != nil {
		return writeAPIError(
			c,
			http.StatusBadRequest,
			types.ErrorCodeInvalidRequest,
			"checkout id must be a positive integer",
		)
	}

	checkout, err := h.service.Return(
		c.Request().Context(),
		checkoutID,
		types.WorkflowMetadata{RequestID: requestID(c)},
	)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusOK, types.CheckoutResponse{Data: checkout})
}

func (h *Checkouts) Get(c *echo.Context) error {
	checkoutID, err := utils.ParseID(c.Param("id"))
	if err != nil {
		return writeAPIError(
			c,
			http.StatusBadRequest,
			types.ErrorCodeInvalidRequest,
			"checkout id must be a positive integer",
		)
	}

	checkout, err := h.service.GetByID(c.Request().Context(), checkoutID)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusOK, types.CheckoutResponse{Data: checkout})
}

func (h *Checkouts) List(c *echo.Context) error {
	pagination, err := parseCheckoutPagination(c)
	if err != nil {
		return writeServiceError(c, err)
	}

	checkouts, metadata, err := h.service.List(c.Request().Context(), pagination)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusOK, types.CheckoutListResponse{
		Data:       checkouts,
		Pagination: metadata,
	})
}

func (h *Checkouts) ListForItem(c *echo.Context) error {
	itemID, err := utils.ParseID(c.Param("id"))
	if err != nil {
		return writeAPIError(
			c,
			http.StatusBadRequest,
			types.ErrorCodeInvalidRequest,
			"item id must be a positive integer",
		)
	}

	pagination, err := parseCheckoutPagination(c)
	if err != nil {
		return writeServiceError(c, err)
	}

	checkouts, metadata, err := h.service.ListByItem(
		c.Request().Context(),
		itemID,
		pagination,
	)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusOK, types.CheckoutListResponse{
		Data:       checkouts,
		Pagination: metadata,
	})
}

func parseCheckoutPagination(c *echo.Context) (types.PaginationRequest, error) {
	limit, err := parseCheckoutQueryInteger(
		c,
		checkoutQueryLimit,
		int64(defaultCheckoutPageLimit),
	)
	if err != nil || limit < 1 || limit > int64(maxCheckoutPageLimit) {
		return types.PaginationRequest{}, types.ErrInvalidPagination
	}

	offset, err := parseCheckoutQueryInteger(c, checkoutQueryOffset, 0)
	// we check that the offset is not negative and does not exceed the maximum value of a 32-bit signed integer.
	if err != nil || offset < 0 || offset > int64(^uint32(0)>>1) {
		return types.PaginationRequest{}, types.ErrInvalidPagination
	}

	return types.PaginationRequest{
		Limit:  int32(limit),
		Offset: int32(offset),
	}, nil
}

func parseCheckoutQueryInteger(
	c *echo.Context,
	name string,
	defaultValue int64,
) (int64, error) {
	values, present := c.QueryParams()[name]
	if !present {
		return defaultValue, nil
	}
	if len(values) != 1 {
		return 0, types.ErrInvalidPagination
	}

	value, err := strconv.ParseInt(values[0], 10, 32)
	if err != nil {
		return 0, types.ErrInvalidPagination
	}
	return value, nil
}
