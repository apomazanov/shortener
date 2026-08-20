// Package config defines types and methods for application configuration.
package config

import (
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/caarlos0/env/v11"
)

// Config contains application configuration parameters.
type Config struct {
	// ServerAddr is this app address.
	ServerAddr string `env:"SERVER_ADDRESS"`
	// BaseURL is a base URL for aliases.
	BaseURL string `env:"BASE_URL"`
	// StorageFile is a path to file for local storage.
	StorageFile string `env:"FILE_STORAGE_PATH"`
	// DatabaseDSN is a database connection path for PostgreSQL storage.
	DatabaseDSN string `env:"DATABASE_DSN"`
	// NoDBMigration is a flag for disabling migration process at application startup.
	NoDBMigration bool `env:"NO_DB_MIGRATION"`
	// JWTSecret is a string for JWT key.
	JWTSecret string `env:"JWT_SECRET"`
	// AuditFile is a path to file for local audit subscriber.
	AuditFile string `env:"AUDIT_FILE"`
	// AuditURL is an address of remote audit subscriber.
	AuditURL string `env:"AUDIT_URL"`
}

// New creates a new config object.
func New(args []string) (*Config, error) {

	cfg := Config{
		ServerAddr: ":8080",
		BaseURL:    "http://localhost:8080",
		JWTSecret:  "",
	}

	// Flags overwrite default values

	fs := flag.NewFlagSet("app_fs", flag.ContinueOnError)

	fs.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "Base URL for aliases")
	fs.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "HTTP-server address:port")
	fs.StringVar(&cfg.StorageFile, "f", cfg.StorageFile, "Storage file path")
	fs.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "Database DSN")
	fs.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "Audit file path")
	fs.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "Audit URL address")

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

	// Validating JWT secret
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT secret not set")
	}

	return &cfg, nil
}

// GetServerAddress returns value of 'ServerAddr' parameter.
func (c *Config) GetServerAddress() string {
	return c.ServerAddr
}

// GetURLBase returns value of 'BaseURL' parameter.
func (c *Config) GetURLBase() string {
	return c.BaseURL
}

// GetStorageFile returns value of 'StorageFile' parameter.
func (c *Config) GetStorageFile() string {
	return c.StorageFile
}

// GetDatabaseDSN returns value of 'DatabaseDSN' parameter.
func (c *Config) GetDatabaseDSN() string {
	return c.DatabaseDSN
}

// GetNoDBMigration returns value of 'NoDBMigration' parameter.
func (c *Config) GetNoDBMigration() bool {
	return c.NoDBMigration
}

// GetJWTSecret returns value of 'JWTSecret' parameter.
func (c *Config) GetJWTSecret() string {
	return c.JWTSecret
}

// GetAuditFile returns value of 'AuditFile' parameter.
func (c *Config) GetAuditFile() string {
	return c.AuditFile
}

// GetAuditURL returns value of 'AuditURL' parameter.
func (c *Config) GetAuditURL() string {
	return c.AuditURL
}
