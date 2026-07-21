package handlers

import (
	"net/http"

	"github.com/0xKa/equipment-checkout-system/server/services"
	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/0xKa/equipment-checkout-system/server/utils"
	"github.com/labstack/echo/v5"
)

type Items struct {
	service services.ItemService
}

func NewItems(service services.ItemService) *Items {
	return &Items{service: service}
}

func (h *Items) Create(c *echo.Context) error {
	var input types.CreateItemRequest
	if err := utils.DecodeJSON(c, &input); err != nil {
		return writeJSONError(c, err)
	}

	item, err := h.service.Create(c.Request().Context(), input)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusCreated, types.ItemResponse{Data: item})
}

func (h *Items) List(c *echo.Context) error {
	items, err := h.service.List(c.Request().Context())
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusOK, types.ItemListResponse{Data: items})
}

func (h *Items) Get(c *echo.Context) error {
	id, err := utils.ParseID(c.Param("id"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "item id must be a positive integer")
	}

	item, err := h.service.GetByID(c.Request().Context(), id)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusOK, types.ItemResponse{Data: item})
}

func (h *Items) Update(c *echo.Context) error {
	id, err := utils.ParseID(c.Param("id"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "item id must be a positive integer")
	}

	var input types.UpdateItemRequest
	if err := utils.DecodeJSON(c, &input); err != nil {
		return writeJSONError(c, err)
	}

	item, err := h.service.Update(c.Request().Context(), id, input)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusOK, types.ItemResponse{Data: item})
}

func (h *Items) Delete(c *echo.Context) error {
	id, err := utils.ParseID(c.Param("id"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "item id must be a positive integer")
	}

	if err := h.service.Delete(c.Request().Context(), id); err != nil {
		return writeServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}
