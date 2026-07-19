package types

type CreateItemRequest struct {
	AssetTag     string `json:"asset_tag"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	SerialNumber string `json:"serial_number"`
}

type UpdateItemRequest = CreateItemRequest
