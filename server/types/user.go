package types

import "time"

// domain
type User struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	Email       *string   `json:"email"`
	DisplayName string    `json:"display_name"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// requests
type CreateUserRequest struct {
	Username    string  `json:"username"`
	Email       *string `json:"email"`
	DisplayName string  `json:"display_name"`
}

type UpdateUserRequest = CreateUserRequest

type UpdateUserStatusRequest struct {
	IsActive *bool `json:"is_active"`
}

// responses
type UserResponse struct {
	Data User `json:"data"`
}

type UserListResponse struct {
	Data []User `json:"data"`
}
