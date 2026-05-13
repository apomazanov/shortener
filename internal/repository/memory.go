package repository

import (
	"context"
	"sync"

	"github.com/apomazanov/shortener/internal/domain"
)

type InMemoryRepo struct {
	data map[string]string
	mu   sync.RWMutex
}

/* -------------------------------------------------------------------------- */
func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{data: make(map[string]string)}
}

/* -------------------------------------------------------------------------- */
func (r *InMemoryRepo) Save(ctx context.Context, alias string, original string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data[alias]; exists {
		return domain.ErrDuplicate
	}

	r.data[alias] = original
	return nil
}

/* -------------------------------------------------------------------------- */
func (r *InMemoryRepo) Get(ctx context.Context, alias string) (original string, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	original, exists := r.data[alias]

	if !exists {
		return "", domain.ErrNotFound
	}

	return original, err
}
