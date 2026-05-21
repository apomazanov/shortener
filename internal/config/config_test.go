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
				ServerAddr:  ":8080",
				BaseURL:     "http://localhost:8080",
				AliasSize:   6,
				StorageFile: "./data/db.json",
			},
			wantErr: false,
		},
		{
			name: "flags override defaults",
			args: []string{"-a", ":9090", "-b", "http://example.com", "-f", "/tmp/test.json"},
			env:  map[string]string{},
			want: &Config{
				ServerAddr:  ":9090",
				BaseURL:     "http://example.com",
				AliasSize:   6,
				StorageFile: "/tmp/test.json",
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
			},
			want: &Config{
				ServerAddr:  ":7070",
				BaseURL:     "http://env.com",
				AliasSize:   6,
				StorageFile: "/env/path.json",
			},
			wantErr: false,
		},
		{
			name:    "invalid base url",
			args:    []string{"-b", "not-a-url"},
			env:     map[string]string{},
			want:    nil,
			wantErr: true,
		},
		{
			name: "trim trailing slash in base url",
			args: []string{"-b", "http://example.com/"},
			env:  map[string]string{},
			want: &Config{
				ServerAddr:  ":8080",
				BaseURL:     "http://example.com",
				AliasSize:   6,
				StorageFile: "./data/db.json",
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
				if cfg.AliasSize != tt.want.AliasSize {
					t.Errorf("AliasSize = %v, want %v", cfg.AliasSize, tt.want.AliasSize)
				}
				if cfg.StorageFile != tt.want.StorageFile {
					t.Errorf("StorageFile = %v, want %v", cfg.StorageFile, tt.want.StorageFile)
				}
			}
		})
	}
}

func TestConfigGetters(t *testing.T) {
	cfg := &Config{
		ServerAddr:  ":1234",
		BaseURL:     "http://test.com",
		AliasSize:   10,
		StorageFile: "/test/file",
	}

	if cfg.GetServerAddress() != ":1234" {
		t.Errorf("GetServerAddress() = %v, want %v", cfg.GetServerAddress(), ":1234")
	}
	if cfg.GetURLBase() != "http://test.com" {
		t.Errorf("GetURLBase() = %v, want %v", cfg.GetURLBase(), "http://test.com")
	}
	if cfg.GetAliasSize() != 10 {
		t.Errorf("GetAliasSize() = %v, want %v", cfg.GetAliasSize(), 10)
	}
	if cfg.GetStorageFile() != "/test/file" {
		t.Errorf("GetStorageFile() = %v, want %v", cfg.GetStorageFile(), "/test/file")
	}
}
