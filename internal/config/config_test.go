package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

/* -------------------------------------------------------------------------- */
func TestConfig_Getters(t *testing.T) {

	cfg := Config{BaseURL: "real base", ServerPort: "true server port"}

	baseURL := cfg.GetURLBase()
	assert.Equal(t, "real base", baseURL)

	serverPort := cfg.GetServerAddress()
	assert.Equal(t, "true server port", serverPort)
}
