package handlers

import (
	"net/http"

	apidocs "github.com/0xKa/equipment-checkout-system/server/api_docs"
	"github.com/0xKa/equipment-checkout-system/server/openapi"
	"github.com/labstack/echo/v5"
)

const scalarContentSecurityPolicy = "default-src 'none'; " +
	"script-src 'self' https://cdn.jsdelivr.net; " +
	"style-src 'unsafe-inline'; " +
	"img-src 'self' data:; " +
	"font-src data:; " +
	"connect-src 'self'; " +
	"worker-src blob:; " +
	"base-uri 'none'; " +
	"form-action 'none'; " +
	"frame-ancestors 'none'"

// APIDocs serves the development API reference and its embedded contract.
type APIDocs struct{}

func NewAPIDocs() *APIDocs {
	return &APIDocs{}
}

func (h *APIDocs) Redirect(c *echo.Context) error {
	return c.Redirect(http.StatusPermanentRedirect, "/scalar/")
}

func (h *APIDocs) Reference(c *echo.Context) error {
	c.Response().Header().Set(
		echo.HeaderContentSecurityPolicy,
		scalarContentSecurityPolicy,
	)
	c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	return c.HTMLBlob(http.StatusOK, apidocs.ScalarPage())
}

func (h *APIDocs) Bootstrap(c *echo.Context) error {
	c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	return c.Blob(
		http.StatusOK,
		"application/javascript; charset=utf-8",
		apidocs.ScalarBootstrap(),
	)
}

func (h *APIDocs) Specification(c *echo.Context) error {
	c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	return c.Blob(
		http.StatusOK,
		"application/yaml; charset=utf-8",
		openapi.Specification(),
	)
}
