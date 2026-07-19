package types

import "errors"

var (
	ErrInvalidInput     = errors.New("invalid input")
	ErrItemNotFound     = errors.New("item not found")
	ErrAssetTagConflict = errors.New("asset tag already exists")
	ErrJSONContentType  = errors.New("content type must be application/json")
	ErrInvalidJSON      = errors.New("invalid JSON body")
)

const (
	ErrorCodeInvalidRequest   = "invalid_request"
	ErrorCodeItemNotFound     = "item_not_found"
	ErrorCodeAssetTagConflict = "asset_tag_conflict"
	ErrorCodeInternal         = "internal_error"
)
