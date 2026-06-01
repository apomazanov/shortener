package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/rs/zerolog"
)

type MemStorage struct {
	mu    sync.RWMutex
	cache map[string]string
}

/* -------------------------------------------------------------------------- */
func NewMemStorage(log *zerolog.Logger) *MemStorage {
	return &MemStorage{cache: make(map[string]string)}
}

/* -------------------------------------------------------------------------- */
func (r *MemStorage) Save(ctx context.Context, alias string, original string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("repo: save aborted: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Searching for alias duplicates first

	if _, exists := r.cache[alias]; exists {
		return domain.ErrDuplicate
	}

	// Appending

	r.cache[alias] = original

	return nil
}

/* -------------------------------------------------------------------------- */
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

/* -------------------------------------------------------------------------- */
func (r *MemStorage) Close() error {
	return nil
}

/* -------------------------------------------------------------------------- */
func (r *MemStorage) Ping(ctx context.Context) error {
	return nil
}
