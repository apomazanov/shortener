// Package config defines types and methods for application configuration.
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
)

// Config contains application configuration parameters.
type Config struct {
	// ServerAddr is this app address.
	ServerAddr string `env:"SERVER_ADDRESS" json:"server_address"`
	// BaseURL is a base URL for aliases.
	BaseURL string `env:"BASE_URL" json:"base_url"`
	// StorageFile is a path to file for local storage.
	StorageFile string `env:"FILE_STORAGE_PATH" json:"file_storage_path"`
	// DatabaseDSN is a database connection path for PostgreSQL storage.
	DatabaseDSN string `env:"DATABASE_DSN" json:"database_dsn"`
	// NoDBMigration is a flag for disabling migration process at application startup.
	NoDBMigration bool `env:"NO_DB_MIGRATION" json:"no_db_migration"`
	// JWTSecret is a string for JWT key.
	JWTSecret string `env:"JWT_SECRET"`
	// AuditFile is a path to file for local audit subscriber.
	AuditFile string `env:"AUDIT_FILE" json:"audit_file"`
	// AuditURL is an address of remote audit subscriber.
	AuditURL string `env:"AUDIT_URL" json:"audit_url"`
	// EnableHTTPS is a flag for HTTPS mode for this server.
	EnableHTTPS bool `env:"ENABLE_HTTPS" envDefault:"false" json:"enable_https"`
	// CertFile is a certificate file for HTTPs.
	CertFile string `env:"CERT_FILE" envDefault:"cert.pem" json:"cert_file"`
	// KeyFile is a key file for HTTPs.
	KeyFile string `env:"KEY_FILE" envDefault:"key.pem" json:"key_file"`
	// ConfigFile is a path to application config file in JSON format.
	ConfigFile string `env:"CONFIG"`
}

func (c *Config) applyNewConfig(new *Config) {
	applyStrValue(&c.ServerAddr, new.ServerAddr)
	applyStrValue(&c.BaseURL, new.BaseURL)
	applyStrValue(&c.StorageFile, new.StorageFile)
	applyStrValue(&c.DatabaseDSN, new.DatabaseDSN)
	applyStrValue(&c.JWTSecret, new.JWTSecret)
	applyStrValue(&c.AuditFile, new.AuditFile)
	applyStrValue(&c.AuditURL, new.AuditURL)
	applyStrValue(&c.CertFile, new.CertFile)
	applyStrValue(&c.KeyFile, new.KeyFile)
	applyStrValue(&c.ConfigFile, new.ConfigFile)

	c.NoDBMigration = new.NoDBMigration
	c.EnableHTTPS = new.EnableHTTPS
}

// New creates a new config object.
func New(args []string) (*Config, error) {

	// Configuration data from flags

	var cfgFlags Config
	fs := flag.NewFlagSet("app_fs", flag.ContinueOnError)

	fs.StringVar(&cfgFlags.BaseURL, "b", cfgFlags.BaseURL, "Base URL for aliases")
	fs.StringVar(&cfgFlags.ServerAddr, "a", cfgFlags.ServerAddr, "HTTP-server address:port")
	fs.StringVar(&cfgFlags.StorageFile, "f", cfgFlags.StorageFile, "Storage file path")
	fs.StringVar(&cfgFlags.DatabaseDSN, "d", cfgFlags.DatabaseDSN, "Database DSN")
	fs.StringVar(&cfgFlags.AuditFile, "audit-file", cfgFlags.AuditFile, "Audit file path")
	fs.StringVar(&cfgFlags.AuditURL, "audit-url", cfgFlags.AuditURL, "Audit URL address")
	fs.BoolVar(&cfgFlags.EnableHTTPS, "s", false, "Enable HTTPS mode")
	fs.StringVar(&cfgFlags.ConfigFile, "c", cfgFlags.ConfigFile, "Config file path")
	fs.StringVar(&cfgFlags.ConfigFile, "config", cfgFlags.ConfigFile, "Config file path")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("flags parsing failed: %w", err)
	}

	// Configuration data from environment variables

	var cfgEnv Config
	if err := env.Parse(&cfgEnv); err != nil {
		return nil, fmt.Errorf("env parsing failed: %w", err)
	}

	// Configuration data from JSON file

	var cfgFileName string

	applyStrValue(&cfgFileName, cfgFlags.ConfigFile)
	applyStrValue(&cfgFileName, cfgEnv.ConfigFile)

	cfgJSON, err := getConfigFromFile(cfgFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Setting configuration data from all sources. Source with higher priority overwrites previous values.

	cfg := Config{
		ServerAddr: ":8080",
		BaseURL:    "http://localhost:8080",
	}

	cfg.applyNewConfig(&cfgJSON)
	cfg.applyNewConfig(&cfgFlags)
	cfg.applyNewConfig(&cfgEnv)

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

// IsHTTPSEnabled shows if HTTPS mode is enabled in config.
func (c *Config) IsHTTPSEnabled() bool {
	return c.EnableHTTPS
}

// GetCertFile returns path to certificate file for HTTPs
func (c *Config) GetCertFile() string {
	return c.CertFile
}

// GetKeyFile returns path to key file for HTTPs
func (c *Config) GetKeyFile() string {
	return c.KeyFile
}

func getConfigFromFile(fileName string) (Config, error) {

	var cfg Config

	if fileName == "" {
		// not an error, returning empty config
		return cfg, nil
	}

	file, err := os.Open(fileName)
	if err != nil {
		return cfg, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func applyStrValue(old *string, new string) {
	if new != "" {
		*old = new
	}
}
