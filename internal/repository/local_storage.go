package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"sync"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/rs/zerolog"
)

// LocalStorage defines data for storage on local hard drive.
type LocalStorage struct {
	// file is a pointer to File object.
	file *os.File
	// encoder makes write operations easier.
	encoder *json.Encoder
	// mu provides thread-safety for data storage
	mu sync.RWMutex
	// cache stores data copy in-memory for fast reading operation and reduction of disc usage.
	cache map[string]string
	// lastUUID is last used UUID value.
	lastUUID int
}

// LocalStorageConfig defines interface for confiduring local storage.
type LocalStorageConfig interface {
	GetStorageFile() string
}

// entry defines data fields of single recoed, stored in local storage.
type entry struct {
	// UUID is a data record ID.
	UUID int `json:"uuid"`
	// Alias is a shortened URL value.
	Alias string `json:"alias"`
	// Original is an original URL value.
	Original string `json:"original"`
}

// NewLocalStorage creates a new local storage object.
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

// Close correctly closes file object of local storage.
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

// Save provides write operation of single recoed to local storage.
func (r *LocalStorage) Save(
	ctx context.Context,
	alias string,
	original string,
	userID string,
) (usedAlias string, err error) {
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

// SaveBatch provides write operation of batch of records to local storage.
func (r *LocalStorage) SaveBatch(
	ctx context.Context,
	toWrite map[string]string,
	userID string,
) (written map[string]string, err error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("repo: batch save aborted: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for alias duplicates to ensure atomicity
	seenInBatch := make(map[string]bool)
	for _, alias := range toWrite {
		if _, exists := r.cache[alias]; exists {
			return nil, domain.ErrAliasDuplicate
		}
		if seenInBatch[alias] {
			return nil, domain.ErrAliasDuplicate
		}
		seenInBatch[alias] = true
	}

	written = make(map[string]string, len(toWrite))

	// Temporary buffer and cache for atomicity of disk i/o operation
	var buf bytes.Buffer
	tempEncoder := json.NewEncoder(&buf)
	tempCache := make(map[string]string)

	isOriginalConflict := false
	currentUUID := r.lastUUID

MainLoop:
	for original, alias := range toWrite {

		// Check original duplicate

		for k, v := range r.cache {
			if v == original {
				written[original] = k
				isOriginalConflict = true
				continue MainLoop
			}
		}

		// Appending

		// Preparing data
		currentUUID++
		entry := entry{UUID: currentUUID, Alias: alias, Original: original}

		// Write to temporary buffer first
		if err := tempEncoder.Encode(entry); err != nil {
			return nil, fmt.Errorf("repo: batch JSON encode error: %w", err)
		}

		tempCache[alias] = original
		written[original] = alias
	}

	// Finalizing write to file
	if _, err := buf.WriteTo(r.file); err != nil {
		return nil, fmt.Errorf("repo: batch file write error: %w", err)
	}

	// Updating actual cache only after successful file write
	maps.Copy(r.cache, tempCache)
	r.lastUUID = currentUUID

	if isOriginalConflict {
		return written, domain.ErrOriginalURLDuplicate
	}

	return written, nil
}

// Get provides a read operation of single record from local storage.
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

// GetByUser returns a list of aliases and original URLs stored by certain user.
func (r *LocalStorage) GetByUser(ctx context.Context, userID string) (data map[string]string, err error) {
	// not implemented for this storage
	return nil, domain.ErrNotFound
}

// Ping is used for detecting database availability.
func (r *LocalStorage) Ping(ctx context.Context) error {
	// not implemented for this storage
	return nil
}

// DeleteBatch provides delete operation for a batch of records from local storage.
func (r *LocalStorage) DeleteBatch(ctx context.Context, batch map[string][]string) error {
	// not implemented for this storage
	return nil
}
