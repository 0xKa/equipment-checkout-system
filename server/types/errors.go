package types

import "errors"

var (
	ErrInvalidInput     = errors.New("invalid input")
	ErrItemNotFound     = errors.New("item not found")
	ErrAssetTagConflict = errors.New("asset tag already exists")
)

const (
	ErrorCodeInvalidRequest   = "invalid_request"
	ErrorCodeItemNotFound     = "item_not_found"
	ErrorCodeAssetTagConflict = "asset_tag_conflict"
	ErrorCodeInternal         = "internal_error"
)
