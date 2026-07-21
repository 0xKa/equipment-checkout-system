package handlers

import (
	"net/http"

	"github.com/0xKa/equipment-checkout-system/server/services"
	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/0xKa/equipment-checkout-system/server/utils"
	"github.com/labstack/echo/v5"
)

type Categories struct {
	service services.CategoryService
}

func NewCategories(service services.CategoryService) *Categories {
	return &Categories{service: service}
}

func (h *Categories) Create(c *echo.Context) error {
	var input types.CreateCategoryRequest
	if err := utils.DecodeJSON(c, &input); err != nil {
		return writeJSONError(c, err)
	}

	category, err := h.service.Create(c.Request().Context(), input)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusCreated, types.CategoryResponse{Data: category})
}

func (h *Categories) List(c *echo.Context) error {
	categories, err := h.service.List(c.Request().Context())
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusOK, types.CategoryListResponse{Data: categories})
}

func (h *Categories) Get(c *echo.Context) error {
	id, err := utils.ParseID(c.Param("id"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "category id must be a positive integer")
	}

	category, err := h.service.GetByID(c.Request().Context(), id)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusOK, types.CategoryResponse{Data: category})
}

func (h *Categories) Update(c *echo.Context) error {
	id, err := utils.ParseID(c.Param("id"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "category id must be a positive integer")
	}

	var input types.UpdateCategoryRequest
	if err := utils.DecodeJSON(c, &input); err != nil {
		return writeJSONError(c, err)
	}

	category, err := h.service.Update(c.Request().Context(), id, input)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusOK, types.CategoryResponse{Data: category})
}

func (h *Categories) Delete(c *echo.Context) error {
	id, err := utils.ParseID(c.Param("id"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "category id must be a positive integer")
	}

	if err := h.service.Delete(c.Request().Context(), id); err != nil {
		return writeServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}
