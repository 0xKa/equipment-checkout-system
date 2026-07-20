package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"

	"github.com/0xKa/equipment-checkout-system/server/services"
	"github.com/0xKa/equipment-checkout-system/server/types"
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
	if err := decodeJSON(c, &input); err != nil {
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
	id, err := parseItemID(c.Param("id"))
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
	id, err := parseItemID(c.Param("id"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "item id must be a positive integer")
	}

	var input types.UpdateItemRequest
	if err := decodeJSON(c, &input); err != nil {
		return writeJSONError(c, err)
	}

	item, err := h.service.Update(c.Request().Context(), id, input)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusOK, types.ItemResponse{Data: item})
}

func (h *Items) Delete(c *echo.Context) error {
	id, err := parseItemID(c.Param("id"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "item id must be a positive integer")
	}

	if err := h.service.Delete(c.Request().Context(), id); err != nil {
		return writeServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

func decodeJSON(c *echo.Context, target any) error {
	mediaType, _, err := mime.ParseMediaType(c.Request().Header.Get(echo.HeaderContentType))
	if err != nil || mediaType != echo.MIMEApplicationJSON {
		return types.ErrInvalidJSON
	}

	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return normalizeJSONError(err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err != nil {
			return normalizeJSONError(err)
		}
		return types.ErrInvalidJSON
	}

	return nil
}

func normalizeJSONError(err error) error {
	if echo.StatusCode(err) == http.StatusRequestEntityTooLarge {
		return err
	}

	return types.ErrInvalidJSON
}

func writeJSONError(c *echo.Context, err error) error {
	if echo.StatusCode(err) == http.StatusRequestEntityTooLarge {
		return err
	}

	return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, err.Error())
}

func parseItemID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, types.ErrInvalidInput
	}

	return id, nil
}

func writeServiceError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, types.ErrInvalidInput):
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "asset_tag and name must not be empty")
	case errors.Is(err, types.ErrItemNotFound):
		return writeAPIError(c, http.StatusNotFound, types.ErrorCodeItemNotFound, "item not found")
	case errors.Is(err, types.ErrAssetTagConflict):
		return writeAPIError(c, http.StatusConflict, types.ErrorCodeAssetTagConflict, "asset_tag already exists")
	default:
		return err
	}
}
