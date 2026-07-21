package utils

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"

	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/labstack/echo/v5"
)

// DecodeJSON decodes one JSON object and rejects unsupported content types,
// unknown fields, and trailing values.
func DecodeJSON(c *echo.Context, target any) error {
	mediaType, _, err := mime.ParseMediaType(c.Request().Header.Get(echo.HeaderContentType))
	if err != nil || mediaType != echo.MIMEApplicationJSON {
		return types.ErrInvalidJSON
	}

	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return normalizeJSONError(err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err != nil {
			return normalizeJSONError(err)
		}
		return types.ErrInvalidJSON
	}

	return nil
}

// ParseID parses a positive base-10 identifier.
func ParseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || !IsValidID(id) {
		return 0, types.ErrInvalidInput
	}

	return id, nil
}

// IsValidID reports whether an identifier is positive.
func IsValidID(id int64) bool {
	return id > 0
}

func normalizeJSONError(err error) error {
	if echo.StatusCode(err) == http.StatusRequestEntityTooLarge {
		return err
	}

	return types.ErrInvalidJSON
}
