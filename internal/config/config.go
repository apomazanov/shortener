package config

type Config struct {
	ServerPort string
	BaseURL    string
}

func (c *Config) GetServerAddress() string {
	return c.ServerPort
}

func (c *Config) GetURLBase() string {
	return c.BaseURL
}
