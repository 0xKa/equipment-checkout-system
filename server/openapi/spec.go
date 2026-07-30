package openapi

import _ "embed"

// specification is embedded so the API runtime image remains a single binary.
//
//go:embed openapi.yaml
var specification []byte

// Specification returns the embedded OpenAPI document.
func Specification() []byte {
	return specification
}
