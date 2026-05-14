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

	// Setup routes
	Setup(e, h)

	// Retrieve registered routes from the Echo instance
	routes := e.Router().Routes()

	foundGet := false
	foundPostText := false
	foundPostJson := false

	for _, r := range routes {
		if r.Method == "GET" && r.Path == "/:short" {
			foundGet = true
		}
		if r.Method == "POST" && r.Path == "/" {
			foundPostText = true
		}
		if r.Method == "POST" && r.Path == "/api/shorten" {
			foundPostJson = true
		}
	}

	assert.True(t, foundGet)
	assert.True(t, foundPostText)
	assert.True(t, foundPostJson)
}
