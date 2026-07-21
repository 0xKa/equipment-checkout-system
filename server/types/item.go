package types

import "time"

const (
	ItemStatusAvailable   = "available"
	ItemStatusCheckedOut  = "checked_out"
	ItemStatusMaintenance = "maintenance"
	ItemStatusRetired     = "retired"
)

// domain
type Item struct {
	ID           int64     `json:"id"`
	CategoryID   *int64    `json:"category_id"`
	AssetTag     string    `json:"asset_tag"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	SerialNumber string    `json:"serial_number"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// requests
type CreateItemRequest struct {
	AssetTag     string `json:"asset_tag"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	SerialNumber string `json:"serial_number"`
	CategoryID   *int64 `json:"category_id"`
}

type UpdateItemRequest = CreateItemRequest

// responses
type ItemResponse struct {
	Data Item `json:"data"`
}

type ItemListResponse struct {
	Data []Item `json:"data"`
}
