package types

import "time"

// Checkout records one borrowing lifecycle for an item.
type Checkout struct {
	ID               int64      `json:"id"`
	ItemID           int64      `json:"item_id"`
	BorrowerUserID   int64      `json:"borrower_user_id"`
	CreatedByUserID  int64      `json:"created_by_user_id"`
	ReturnedToUserID *int64     `json:"returned_to_user_id"`
	CheckedOutAt     time.Time  `json:"checked_out_at"`
	DueAt            *time.Time `json:"due_at"`
	ReturnedAt       *time.Time `json:"returned_at"`
	Notes            string     `json:"notes"`
}

// CreateCheckoutRequest contains the borrower-controlled checkout fields.
type CreateCheckoutRequest struct {
	BorrowerUserID int64      `json:"borrower_user_id"`
	DueAt          *time.Time `json:"due_at"`
	Notes          string     `json:"notes"`
}

// WorkflowMetadata carries request correlation data into audit writes.
type WorkflowMetadata struct {
	RequestID string
}

// PaginationRequest is the validated bounded checkout list window.
type PaginationRequest struct {
	Limit  int32
	Offset int32
}

// PaginationMetadata describes one checkout list page.
type PaginationMetadata struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
	Count  int   `json:"count"`
	Total  int64 `json:"total"`
}

type CheckoutResponse struct {
	Data Checkout `json:"data"`
}

type CheckoutListResponse struct {
	Data       []Checkout         `json:"data"`
	Pagination PaginationMetadata `json:"pagination"`
}
