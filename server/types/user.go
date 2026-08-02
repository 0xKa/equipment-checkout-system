package types

import "time"

type UserRole string

const (
	UserRoleEmployee       UserRole = "employee"
	UserRoleInventoryAdmin UserRole = "inventory_admin"
	UserRoleAuditor        UserRole = "auditor"
)

func (r UserRole) Valid() bool {
	switch r {
	case UserRoleEmployee, UserRoleInventoryAdmin, UserRoleAuditor:
		return true
	default:
		return false
	}
}

// domain
type User struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	Email       *string   `json:"email"`
	DisplayName string    `json:"display_name"`
	Role        UserRole  `json:"role"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// requests
type CreateUserRequest struct {
	Username    string   `json:"username"`
	Email       *string  `json:"email"`
	DisplayName string   `json:"display_name"`
	Role        UserRole `json:"role"`
}

type UpdateUserProfileRequest struct {
	Username    string  `json:"username"`
	Email       *string `json:"email"`
	DisplayName string  `json:"display_name"`
}

type UpdateUserRoleRequest struct {
	Role UserRole `json:"role"`
}

type UpdateUserStatusRequest struct {
	IsActive *bool `json:"is_active"`
}

type SetTemporaryPasswordRequest struct {
	Password string `json:"password"`
}

// responses
type UserResponse struct {
	Data User `json:"data"`
}

type UserListResponse struct {
	Data []User `json:"data"`
}
