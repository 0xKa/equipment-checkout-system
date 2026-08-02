package handlers

import (
	"net/http"

	"github.com/0xKa/equipment-checkout-system/server/services"
	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/0xKa/equipment-checkout-system/server/utils"
	"github.com/labstack/echo/v5"
)

type Users struct {
	service services.UserService
}

// Builds the user HTTP handlers.
func NewUsers(service services.UserService) *Users {
	return &Users{service: service}
}

// Starts user creation, delegates Keycloak/PostgreSQL synchronization, and
// returns the linked user.
func (h *Users) Create(c *echo.Context) error {
	var input types.CreateUserRequest
	if err := utils.DecodeJSON(c, &input); err != nil {
		return writeJSONError(c, err)
	}

	user, err := h.service.Create(c.Request().Context(), input)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusCreated, types.UserResponse{Data: user})
}

// Lists users with an optional active-state filter.
func (h *Users) List(c *echo.Context) error {
	isActive, err := parseActiveFilter(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "is_active must be true or false")
	}

	users, err := h.service.List(c.Request().Context(), isActive)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusOK, types.UserListResponse{Data: users})
}

// Returns the user identified by the path ID.
func (h *Users) Get(c *echo.Context) error {
	id, err := utils.ParseID(c.Param("id"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "user id must be a positive integer")
	}

	user, err := h.service.GetByID(c.Request().Context(), id)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusOK, types.UserResponse{Data: user})
}

// Replaces editable profile fields without changing status.
func (h *Users) Update(c *echo.Context) error {
	id, err := utils.ParseID(c.Param("id"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "user id must be a positive integer")
	}

	var input types.UpdateUserProfileRequest
	if err := utils.DecodeJSON(c, &input); err != nil {
		return writeJSONError(c, err)
	}

	user, err := h.service.UpdateProfile(c.Request().Context(), id, input)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusOK, types.UserResponse{Data: user})
}

// Changes the user's single application role.
func (h *Users) SetRole(c *echo.Context) error {
	id, err := utils.ParseID(c.Param("id"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "user id must be a positive integer")
	}

	var input types.UpdateUserRoleRequest
	if err := utils.DecodeJSON(c, &input); err != nil {
		return writeJSONError(c, err)
	}

	user, err := h.service.SetRole(c.Request().Context(), id, input.Role)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusOK, types.UserResponse{Data: user})
}

// Activates or deactivates a user.
func (h *Users) SetStatus(c *echo.Context) error {
	id, err := utils.ParseID(c.Param("id"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "user id must be a positive integer")
	}

	var input types.UpdateUserStatusRequest
	if err := utils.DecodeJSON(c, &input); err != nil {
		return writeJSONError(c, err)
	}
	if input.IsActive == nil {
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "is_active is required")
	}

	user, err := h.service.SetActive(c.Request().Context(), id, *input.IsActive)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(http.StatusOK, types.UserResponse{Data: user})
}

// Disables the linked identity while preserving the local user and history.
func (h *Users) Deprovision(c *echo.Context) error {
	id, err := utils.ParseID(c.Param("id"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "user id must be a positive integer")
	}

	if err := h.service.Deprovision(c.Request().Context(), id); err != nil {
		return writeServiceError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// Sends a temporary credential directly to Keycloak without returning it.
func (h *Users) SetTemporaryPassword(c *echo.Context) error {
	id, err := utils.ParseID(c.Param("id"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, types.ErrorCodeInvalidRequest, "user id must be a positive integer")
	}

	var input types.SetTemporaryPasswordRequest
	if err := utils.DecodeJSON(c, &input); err != nil {
		return writeJSONError(c, err)
	}
	if err := h.service.SetTemporaryPassword(
		c.Request().Context(), id, input.Password,
	); err != nil {
		return writeServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// Returns the active actor stored in the request context.
func (h *Users) Me(c *echo.Context) error {
	actor, ok := types.ActorFromContext(c.Request().Context())
	if !ok {
		return writeServiceError(c, types.ErrAuthenticationRequired)
	}

	return c.JSON(http.StatusOK, types.UserResponse{Data: actor.User})
}

// Parses an optional true or false active-state filter.
func parseActiveFilter(c *echo.Context) (*bool, error) {
	values, present := c.QueryParams()["is_active"]
	if !present {
		return nil, nil
	}
	if len(values) != 1 {
		return nil, types.ErrInvalidInput
	}

	switch values[0] {
	case "true":
		value := true
		return &value, nil
	case "false":
		value := false
		return &value, nil
	default:
		return nil, types.ErrInvalidInput
	}
}
