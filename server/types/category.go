package types

import "time"

// domain
type Category struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// requests
type CreateCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateCategoryRequest = CreateCategoryRequest

// responses
type CategoryResponse struct {
	Data Category `json:"data"`
}

type CategoryListResponse struct {
	Data []Category `json:"data"`
}
