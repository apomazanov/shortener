package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/rs/zerolog"
)

type LocalStorage struct {
	file     *os.File
	encoder  *json.Encoder
	mu       sync.RWMutex
	cache    map[string]string
	lastUUID int
}

type LocalStorageConfig interface {
	GetStorageFile() string
}

type entry struct {
	UUID     int    `json:"uuid"`
	Alias    string `json:"alias"`
	Original string `json:"original"`
}

/* -------------------------------------------------------------------------- */
func NewLocalStorage(cfg LocalStorageConfig, log *zerolog.Logger) (*LocalStorage, error) {

	storage := &LocalStorage{cache: make(map[string]string)}

	// Create storage file, if not exists

	filename := cfg.GetStorageFile()

	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("repo: cannot create directories: %w", err)
	}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("repo: cannot create/open file: %w", err)
	}

	// Read existing file (empty, if new) to fill cache

	decoder := json.NewDecoder(file)
	lineNum := 0
	for {
		lineNum++

		var e entry
		err = decoder.Decode(&e)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			log.Warn().
				Int("error_line", lineNum).
				Msg("parsing error")

			// bad record is just passed by, not interrupting
			continue
		}

		storage.cache[e.Alias] = e.Original

		if storage.lastUUID < e.UUID {
			storage.lastUUID = e.UUID
		}
	}

	storage.file = file
	storage.encoder = json.NewEncoder(file)

	return storage, nil
}

/* -------------------------------------------------------------------------- */
func (r *LocalStorage) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	errSync := r.file.Sync()
	errClose := r.file.Close()

	if errSync != nil {
		return fmt.Errorf("repo: sync error: %w", errSync)
	}

	if errClose != nil {
		return fmt.Errorf("repo: close error: %w", errClose)
	}

	return nil
}

/* -------------------------------------------------------------------------- */
func (r *LocalStorage) Save(ctx context.Context, alias string, original string) (usedAlias string, err error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("repo: save aborted: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Searching for original URL duplicates first

	for k, v := range r.cache {
		if v == original {
			return k, domain.ErrOriginalURLDuplicate
		}
	}

	// Searching for alias duplicates

	if _, exists := r.cache[alias]; exists {
		return "", domain.ErrAliasDuplicate
	}

	// Appending

	// Preparing data
	newLastUUID := r.lastUUID + 1
	entry := entry{UUID: newLastUUID, Alias: alias, Original: original}

	// Writing
	if err := r.encoder.Encode(entry); err != nil {
		return "", fmt.Errorf("repo: JSON encode error: %w", err)
	}

	// Updating cache, if data write was successful
	r.cache[alias] = original
	r.lastUUID = newLastUUID

	return alias, nil
}

/* -------------------------------------------------------------------------- */
func (r *LocalStorage) Get(ctx context.Context, alias string) (original string, err error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("repo: get aborted: %w", err)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	original, exists := r.cache[alias]
	if exists {
		return original, nil
	}

	return "", domain.ErrNotFound
}

/* -------------------------------------------------------------------------- */
func (r *LocalStorage) Ping(ctx context.Context) error {
	return nil
}
