package types

import "time"

const (
	ItemStatusAvailable   = "available"
	ItemStatusCheckedOut  = "checked_out"
	ItemStatusMaintenance = "maintenance"
	ItemStatusRetired     = "retired"
)

type Item struct {
	ID           int64     `json:"id"`
	AssetTag     string    `json:"asset_tag"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	SerialNumber string    `json:"serial_number"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
