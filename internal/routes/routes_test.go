package routes

import (
	"testing"

	"github.com/apomazanov/shortener/internal/handlers"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

/* -------------------------------------------------------------------------- */
func TestSetup(t *testing.T) {
	// Initialize Echo and the handler
	e := echo.New()
	h := &handlers.Handler{}

	// Dummy auth middleware for testing
	dummyMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	}

	// Setup routes
	Setup(e, h, dummyMiddleware)

	// Retrieve registered routes from the Echo instance
	routes := e.Router().Routes()

	// List of all expected endpoints in the routes configuration
	expectedRoutes := []struct {
		method string
		path   string
	}{
		{method: "GET", path: "/:alias"},
		{method: "GET", path: "/ping"},
		{method: "GET", path: "/api/user/urls"},
		{method: "POST", path: "/"},
		{method: "POST", path: "/api/shorten"},
		{method: "POST", path: "/api/shorten/batch"},
		{method: "DELETE", path: "/api/user/urls"},
	}

	// Assert that each expected endpoint is correctly registered
	for _, expected := range expectedRoutes {
		found := false
		for _, r := range routes {
			if r.Method == expected.method && r.Path == expected.path {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected route %s %s to be registered", expected.method, expected.path)
	}
}
