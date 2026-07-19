package types

type HealthResponse struct {
	Status string `json:"status"`
}

type ItemResponse struct {
	Data Item `json:"data"`
}

type ItemListResponse struct {
	Data []Item `json:"data"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}
