package types

import "errors"

// errors
var (
	ErrInvalidInput         = errors.New("invalid input")
	ErrInvalidCategoryID    = errors.New("invalid category id")
	ErrInvalidCategoryInput = errors.New("invalid category input")
	ErrItemNotFound         = errors.New("item not found")
	ErrAssetTagConflict     = errors.New("asset tag already exists")
	ErrSerialNumberConflict = errors.New("serial number already exists")
	ErrCategoryNotFound     = errors.New("category not found")
	ErrCategoryNameConflict = errors.New("category name already exists")
	ErrCategoryInUse        = errors.New("category is in use")
	ErrItemInUse            = errors.New("item is in use")
	ErrInvalidUserID        = errors.New("invalid user id")
	ErrInvalidUserInput     = errors.New("invalid user input")
	ErrUserNotFound         = errors.New("user not found")
	ErrUsernameConflict     = errors.New("username already exists")
	ErrUserEmailConflict    = errors.New("user email already exists")
	ErrActorRequired        = errors.New("actor user id is required")
	ErrInvalidActor         = errors.New("invalid actor user id")
	ErrActorInactive        = errors.New("actor is inactive")
	ErrJSONContentType      = errors.New("content type must be application/json")
	ErrInvalidJSON          = errors.New("invalid JSON body")
)

// error codes
const (
	ErrorCodeInvalidRequest       = "invalid_request"
	ErrorCodeItemNotFound         = "item_not_found"
	ErrorCodeAssetTagConflict     = "asset_tag_conflict"
	ErrorCodeSerialNumberConflict = "serial_number_conflict"
	ErrorCodeCategoryNotFound     = "category_not_found"
	ErrorCodeCategoryNameConflict = "category_name_conflict"
	ErrorCodeCategoryInUse        = "category_in_use"
	ErrorCodeItemInUse            = "item_in_use"
	ErrorCodeUserNotFound         = "user_not_found"
	ErrorCodeUsernameConflict     = "username_conflict"
	ErrorCodeUserEmailConflict    = "email_conflict"
	ErrorCodeInvalidActor         = "invalid_actor"
	ErrorCodeActorInactive        = "actor_inactive"
	ErrorCodeInternal             = "internal_error"
)

// responses
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}
