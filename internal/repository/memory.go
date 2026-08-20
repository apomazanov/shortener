package repository

import (
	"context"
	"fmt"
	"maps"
	"sync"

	"github.com/apomazanov/shortener/internal/domain"
)

// MemStorage defines data for in-memory storage.
type MemStorage struct {
	// mu is a mutex for in-memory thread-safety.
	mu sync.RWMutex
	// cache contains aliases and original URLs.
	cache map[string]string
}

// NewMemStorage creates a new MemStorage object.
func NewMemStorage() *MemStorage {
	return &MemStorage{cache: make(map[string]string)}
}

// Save provides write operation of single recoed to in-memory storage.
func (r *MemStorage) Save(
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

	r.cache[alias] = original

	return alias, nil
}

// SaveBatch provides write operation of batch of records to in-memory storage.
func (r *MemStorage) SaveBatch(
	ctx context.Context,
	toWrite map[string]string,
	userID string,
) (written map[string]string, err error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("repo: batch save aborted: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for alias duplicates (in cache and within the batch) to ensure atomicity
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
	tempCache := make(map[string]string)
	isOriginalConflict := false

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

		tempCache[alias] = original
		written[original] = alias
	}

	maps.Copy(r.cache, tempCache)

	if isOriginalConflict {
		return written, domain.ErrOriginalURLDuplicate
	}

	return written, nil
}

// Get provides a read operation of single record from in-memory storage.
func (r *MemStorage) Get(ctx context.Context, alias string) (original string, err error) {
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
func (r *MemStorage) GetByUser(ctx context.Context, userID string) (data map[string]string, err error) {
	return nil, domain.ErrNotFound
}

// Close correctly closes file object of in-memory storage.
func (r *MemStorage) Close() error {
	return nil
}

// Ping is used for detecting database availability.
func (r *MemStorage) Ping(ctx context.Context) error {
	return nil
}

// DeleteBatch provides delete operation for a batch of records from in-memory storage.
func (r *MemStorage) DeleteBatch(ctx context.Context, batch map[string][]string) error {
	return nil
}
