package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/* -------------------------------------------------------------------------- */
func TestConfigGetters(t *testing.T) {

	cfg := Config{BaseURL: "real base", ServerAddr: "true server port", AliasSize: 3, RepoFile: "./custom.json"}

	baseURL := cfg.GetUrlBase()
	assert.Equal(t, "real base", baseURL)

	serverPort := cfg.GetServerAddress()
	assert.Equal(t, "true server port", serverPort)

	aliasSize := cfg.GetAliasSize()
	assert.Equal(t, 3, aliasSize)

	repoFile := cfg.GetRepoFile()
	assert.Equal(t, "./custom.json", repoFile)
}

/* -------------------------------------------------------------------------- */
func TestConfigNew_Default(t *testing.T) {
	args := []string{}

	cfg, err := New(args)

	assert.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, ":8080", cfg.ServerAddr)
	assert.Equal(t, "http://localhost:8080", cfg.BaseURL)
	assert.Equal(t, 6, cfg.AliasSize)
	assert.Equal(t, "./data/db.json", cfg.RepoFile)
}

/* -------------------------------------------------------------------------- */
func TestConfigNew_Flags(t *testing.T) {
	args := []string{"-b", "http://fl.ag", "-a", ":9999", "-f", "./custom.json"}

	cfg, err := New(args)

	assert.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, ":9999", cfg.ServerAddr)
	assert.Equal(t, "http://fl.ag", cfg.BaseURL)
	assert.Equal(t, "./custom.json", cfg.RepoFile)
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
	t.Setenv("ALIAS_SIZE", "7")
	t.Setenv("REPO_FILE", "./custom.json")

	cfg, err := New(args)

	assert.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, ":1001", cfg.ServerAddr)
	assert.Equal(t, "http://en.v", cfg.BaseURL)
	assert.Equal(t, 7, cfg.AliasSize)
	assert.Equal(t, "./custom.json", cfg.RepoFile)
}
