package types

import "errors"

// errors
var (
	ErrInvalidInput           = errors.New("invalid input")
	ErrInvalidCategoryID      = errors.New("invalid category id")
	ErrInvalidCategoryInput   = errors.New("invalid category input")
	ErrItemNotFound           = errors.New("item not found")
	ErrAssetTagConflict       = errors.New("asset tag already exists")
	ErrSerialNumberConflict   = errors.New("serial number already exists")
	ErrCategoryNotFound       = errors.New("category not found")
	ErrCategoryNameConflict   = errors.New("category name already exists")
	ErrCategoryInUse          = errors.New("category is in use")
	ErrItemInUse              = errors.New("item is in use")
	ErrInvalidUserID          = errors.New("invalid user id")
	ErrInvalidUserInput       = errors.New("invalid user input")
	ErrUserNotFound           = errors.New("user not found")
	ErrUsernameConflict       = errors.New("username already exists")
	ErrUserEmailConflict      = errors.New("user email already exists")
	ErrIdentityNotLinked      = errors.New("external identity is not linked")
	ErrAuthenticationRequired = errors.New("authentication is required")
	ErrInvalidToken           = errors.New("invalid access token")
	ErrForbidden              = errors.New("access is forbidden")
	ErrAccountInactive        = errors.New("account is inactive")
	ErrInvalidBorrowerID      = errors.New("invalid borrower user id")
	ErrBorrowerNotFound       = errors.New("borrower user not found")
	ErrBorrowerInactive       = errors.New("borrower is inactive")
	ErrInvalidCheckoutID      = errors.New("invalid checkout id")
	ErrInvalidCheckoutDueAt   = errors.New("invalid checkout due date")
	ErrCheckoutNotFound       = errors.New("checkout not found")
	ErrItemNotAvailable       = errors.New("item is not available")
	ErrCheckoutReturned       = errors.New("checkout is already returned")
	ErrCheckoutStateConflict  = errors.New("checkout and item state conflict")
	ErrInvalidPagination      = errors.New("invalid pagination")
	ErrJSONContentType        = errors.New("content type must be application/json")
	ErrInvalidJSON            = errors.New("invalid JSON body")
)

// error codes
const (
	ErrorCodeInvalidRequest         = "invalid_request"
	ErrorCodeItemNotFound           = "item_not_found"
	ErrorCodeAssetTagConflict       = "asset_tag_conflict"
	ErrorCodeSerialNumberConflict   = "serial_number_conflict"
	ErrorCodeCategoryNotFound       = "category_not_found"
	ErrorCodeCategoryNameConflict   = "category_name_conflict"
	ErrorCodeCategoryInUse          = "category_in_use"
	ErrorCodeItemInUse              = "item_in_use"
	ErrorCodeUserNotFound           = "user_not_found"
	ErrorCodeUsernameConflict       = "username_conflict"
	ErrorCodeUserEmailConflict      = "email_conflict"
	ErrorCodeAuthenticationRequired = "authentication_required"
	ErrorCodeInvalidToken           = "invalid_token"
	ErrorCodeForbidden              = "forbidden"
	ErrorCodeIdentityNotLinked      = "identity_not_linked"
	ErrorCodeAccountInactive        = "account_inactive"
	ErrorCodeBorrowerInactive       = "borrower_inactive"
	ErrorCodeCheckoutNotFound       = "checkout_not_found"
	ErrorCodeItemNotAvailable       = "item_not_available"
	ErrorCodeCheckoutReturned       = "checkout_already_returned"
	ErrorCodeCheckoutStateConflict  = "checkout_state_conflict"
	ErrorCodeRequestTimeout         = "request_timeout"
	ErrorCodeServiceUnavailable     = "service_unavailable"
	ErrorCodeInternal               = "internal_error"
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
