package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/* -------------------------------------------------------------------------- */
func TestConfigGetters(t *testing.T) {

	cfg := Config{BaseURL: "real base", ServerAddr: "true server port", AliasSize: 3}

	baseURL := cfg.GetUrlBase()
	assert.Equal(t, "real base", baseURL)

	serverPort := cfg.GetServerAddress()
	assert.Equal(t, "true server port", serverPort)

	aliasSize := cfg.GetAliasSize()
	assert.Equal(t, 3, aliasSize)
}

/* -------------------------------------------------------------------------- */
func TestConfigNew_Default(t *testing.T) {
	args := []string{}

	cfg, err := New(args)

	assert.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, ":8080", cfg.ServerAddr)
	assert.Equal(t, "http://localhost:8080", cfg.BaseURL)
}

/* -------------------------------------------------------------------------- */
func TestConfigNew_Flags(t *testing.T) {
	args := []string{"-b", "http://fl.ag", "-a", ":9999"}

	cfg, err := New(args)

	assert.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, ":9999", cfg.ServerAddr)
	assert.Equal(t, "http://fl.ag", cfg.BaseURL)
}

/* -------------------------------------------------------------------------- */
func TestConfigNew_FlagsError(t *testing.T) {
	args := []string{"-c", "trash", "-d", ":trash"}

	cfg, err := New(args)

	assert.Error(t, err)
	assert.Nil(t, cfg)
}

/* -------------------------------------------------------------------------- */
func TestConfigNew_Env(t *testing.T) {
	args := []string{"-b", "http://fl.ag", "-a", ":9999"}

	t.Setenv("SERVER_ADDRESS", ":1001")
	t.Setenv("BASE_URL", "http://en.v")

	cfg, err := New(args)

	assert.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, ":1001", cfg.ServerAddr)
	assert.Equal(t, "http://en.v", cfg.BaseURL)
}
