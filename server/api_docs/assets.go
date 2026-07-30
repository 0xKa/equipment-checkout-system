package apidocs

import _ "embed"

//go:embed scalar.html
var scalarPage []byte

//go:embed bootstrap.js
var scalarBootstrap []byte

// ScalarPage returns the embedded API reference page.
func ScalarPage() []byte {
	return scalarPage
}

// ScalarBootstrap returns the embedded Scalar configuration script.
func ScalarBootstrap() []byte {
	return scalarBootstrap
}
