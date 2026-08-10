package config

import (
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	ServerAddr    string `env:"SERVER_ADDRESS"`
	BaseURL       string `env:"BASE_URL"`
	StorageFile   string `env:"FILE_STORAGE_PATH"`
	DatabaseDSN   string `env:"DATABASE_DSN"`
	NoDBMigration bool   `env:"NO_DB_MIGRATION"`
	JWTSecret     string `env:"JWT_SECRET"`
	AuditFile     string `env:"AUDIT_FILE"`
	AuditURL      string `env:"AUDIT_URL"`
}

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

func (c *Config) GetServerAddress() string {
	return c.ServerAddr
}

func (c *Config) GetURLBase() string {
	return c.BaseURL
}

func (c *Config) GetStorageFile() string {
	return c.StorageFile
}

func (c *Config) GetDatabaseDSN() string {
	return c.DatabaseDSN
}

func (c *Config) GetNoDBMigration() bool {
	return c.NoDBMigration
}

func (c *Config) GetJWTSecret() string {
	return c.JWTSecret
}

func (c *Config) GetAuditFile() string {
	return c.AuditFile
}

func (c *Config) GetAuditURL() string {
	return c.AuditURL
}
