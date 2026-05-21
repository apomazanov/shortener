package config

import (
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	ServerAddr  string `env:"SERVER_ADDRESS"`
	BaseURL     string `env:"BASE_URL"`
	AliasSize   int    `env:"ALIAS_SIZE"`
	StorageFile string `env:"FILE_STORAGE_PATH"`
}

/* -------------------------------------------------------------------------- */
func New(args []string) (*Config, error) {

	cfg := Config{
		ServerAddr:  ":8080",
		BaseURL:     "http://localhost:8080",
		AliasSize:   6,
		StorageFile: "./data/db.json",
	}

	// Flags overwrite default values

	fs := flag.NewFlagSet("app_fs", flag.ContinueOnError)

	fs.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "Base URL for aliases")
	fs.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "HTTP-server address:port")
	fs.StringVar(&cfg.StorageFile, "f", cfg.StorageFile, "Storage file path")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("flags parsing failed: %w", err)
	}

	// Environment variables have highest priority

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("env parsing failed: %w", err)
	}

	// Validating BaseURL
	if _, err := url.ParseRequestURI(cfg.BaseURL); err != nil {
		return nil, fmt.Errorf("invalid BaseURL: %w", err)
	}
	cfg.BaseURL = strings.TrimSuffix(cfg.BaseURL, "/")

	return &cfg, nil
}

/* -------------------------------------------------------------------------- */
func (c *Config) GetServerAddress() string {
	return c.ServerAddr
}

/* -------------------------------------------------------------------------- */
func (c *Config) GetURLBase() string {
	return c.BaseURL
}

/* -------------------------------------------------------------------------- */
func (c *Config) GetAliasSize() int {
	return c.AliasSize
}

/* -------------------------------------------------------------------------- */
func (c *Config) GetStorageFile() string {
	return c.StorageFile
}
