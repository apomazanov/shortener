package config

import (
	"flag"

	"github.com/caarlos0/env"
)

type Config struct {
	ServerAddr string	`env:"SERVER_ADDRESS"`
	BaseURL    string	`env:"BASE_URL"`
}

func New(args []string) (*Config, error) {
	// Default values are set here

	cfg := Config{
		ServerAddr: ":8080",
		BaseURL: "http://localhost:8080",
	}

	// Flags overwrite default values

	fs := flag.NewFlagSet("app_fs", flag.ContinueOnError)

	fs.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "Base URL for aliases")
	fs.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "HTTP-server address:port")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// Environment variables have highest priority

	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) GetServerAddress() string {
	return c.ServerAddr
}

func (c *Config) GetURLBase() string {
	return c.BaseURL
}
