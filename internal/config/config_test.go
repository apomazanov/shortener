package config

import (
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	clearEnv := func() {
		os.Unsetenv("SERVER_ADDRESS")
		os.Unsetenv("BASE_URL")
		os.Unsetenv("FILE_STORAGE_PATH")
		os.Unsetenv("ALIAS_SIZE")
		os.Unsetenv("DATABASE_DSN")
		os.Unsetenv("NO_DB_MIGRATION")
		os.Unsetenv("JWT_SECRET")
	}

	tests := []struct {
		name    string
		args    []string
		env     map[string]string
		want    *Config
		wantErr bool
	}{
		{
			name: "default values",
			args: []string{},
			env:  map[string]string{},
			want: &Config{
				ServerAddr:    ":8080",
				BaseURL:       "http://localhost:8080",
				StorageFile:   "",
				DatabaseDSN:   "",
				NoDBMigration: false,
				JWTSecret:     "",
			},
			wantErr: true,
		},
		{
			name: "flags override defaults",
			args: []string{"-a", ":9090", "-b", "http://example.com", "-f", "/tmp/test.json", "-d", "postgres://user:pass@localhost:5432/db"},
			env: map[string]string{
				"JWT_SECRET": "secure key",
			},
			want: &Config{
				ServerAddr:    ":9090",
				BaseURL:       "http://example.com",
				StorageFile:   "/tmp/test.json",
				DatabaseDSN:   "postgres://user:pass@localhost:5432/db",
				NoDBMigration: false,
			},
			wantErr: false,
		},
		{
			name: "env variables override flags",
			args: []string{"-a", ":9090"},
			env: map[string]string{
				"SERVER_ADDRESS":    ":7070",
				"BASE_URL":          "http://env.com",
				"FILE_STORAGE_PATH": "/env/path.json",
				"DATABASE_DSN":      "postgres://env:env@localhost:5432/env_db",
				"JWT_SECRET":        "secure key",
			},
			want: &Config{
				ServerAddr:    ":7070",
				BaseURL:       "http://env.com",
				StorageFile:   "/env/path.json",
				DatabaseDSN:   "postgres://env:env@localhost:5432/env_db",
				NoDBMigration: false,
				JWTSecret:     "secure key",
			},
			wantErr: false,
		},
		{
			name: "env variable NO_DB_MIGRATION",
			args: []string{},
			env: map[string]string{
				"NO_DB_MIGRATION": "true",
				"JWT_SECRET":      "secure key",
			},
			want: &Config{
				ServerAddr:    ":8080",
				BaseURL:       "http://localhost:8080",
				StorageFile:   "",
				DatabaseDSN:   "",
				NoDBMigration: true,
			},
			wantErr: false,
		},
		{
			name: "invalid base url",
			args: []string{"-b", "not-a-url"},
			env: map[string]string{
				"JWT_SECRET": "secure key",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty JWT secret",
			args:    []string{"-a", ":9090"},
			env:     map[string]string{},
			want:    nil,
			wantErr: true,
		},
		{
			name: "trim trailing slash in base url",
			args: []string{"-b", "http://example.com/"},
			env: map[string]string{
				"JWT_SECRET": "secure key",
			},
			want: &Config{
				ServerAddr:    ":8080",
				BaseURL:       "http://example.com",
				StorageFile:   "",
				DatabaseDSN:   "",
				NoDBMigration: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv()
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			cfg, err := New(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if cfg.ServerAddr != tt.want.ServerAddr {
					t.Errorf("ServerAddr = %v, want %v", cfg.ServerAddr, tt.want.ServerAddr)
				}
				if cfg.BaseURL != tt.want.BaseURL {
					t.Errorf("BaseURL = %v, want %v", cfg.BaseURL, tt.want.BaseURL)
				}
				if cfg.StorageFile != tt.want.StorageFile {
					t.Errorf("StorageFile = %v, want %v", cfg.StorageFile, tt.want.StorageFile)
				}
				if cfg.DatabaseDSN != tt.want.DatabaseDSN {
					t.Errorf("DatabaseDSN = %v, want %v", cfg.DatabaseDSN, tt.want.DatabaseDSN)
				}
				if cfg.NoDBMigration != tt.want.NoDBMigration {
					t.Errorf("NoDBMigration = %v, want %v", cfg.NoDBMigration, tt.want.NoDBMigration)
				}
			}
		})
	}
}

func TestConfigGetters(t *testing.T) {
	cfg := &Config{
		ServerAddr:    ":1234",
		BaseURL:       "http://test.com",
		StorageFile:   "/test/file",
		DatabaseDSN:   "postgres://localhost:5432/test",
		NoDBMigration: true,
		JWTSecret:     "secure key",
	}

	if cfg.GetServerAddress() != ":1234" {
		t.Errorf("GetServerAddress() = %v, want %v", cfg.GetServerAddress(), ":1234")
	}
	if cfg.GetURLBase() != "http://test.com" {
		t.Errorf("GetURLBase() = %v, want %v", cfg.GetURLBase(), "http://test.com")
	}
	if cfg.GetStorageFile() != "/test/file" {
		t.Errorf("GetStorageFile() = %v, want %v", cfg.GetStorageFile(), "/test/file")
	}
	if cfg.GetDatabaseDSN() != "postgres://localhost:5432/test" {
		t.Errorf("GetDatabaseDSN() = %v, want %v", cfg.GetDatabaseDSN(), "postgres://localhost:5432/test")
	}
	if cfg.GetNoDBMigration() != true {
		t.Errorf("GetNoDBMigration() = %v, want %v", cfg.GetNoDBMigration(), true)
	}
	if cfg.GetJWTSecret() != "secure key" {
		t.Errorf("GetJWTSecret() = %v, want %v", cfg.GetNoDBMigration(), "secure key")
	}
}
