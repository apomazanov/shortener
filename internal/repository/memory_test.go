package repository

import (
	"context"
	"testing"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/stretchr/testify/assert"
)

/* -------------------------------------------------------------------------- */
func TestNewInMemoryRepo(t *testing.T) {
	repo := NewInMemoryRepo()
	assert.NotNil(t, repo)
	assert.NotNil(t, repo.data)
}

/* -------------------------------------------------------------------------- */
func TestInMemoryRepo_Save(t *testing.T) {
	repo := NewInMemoryRepo()
	ctx := context.Background()

	t.Run("save successful", func(t *testing.T) {
		alias := "short1"
		original := "http://google.com"

		err := repo.Save(ctx, alias, original)

		assert.NoError(t, err)
		assert.Equal(t, original, repo.data[alias])
	})

	t.Run("save duplicate", func(t *testing.T) {
		alias := "dup"
		repo.data[alias] = "http://first.com"

		err := repo.Save(ctx, alias, "http://second.com")

		assert.ErrorIs(t, err, domain.ErrDuplicate)
		assert.Equal(t, "http://first.com", repo.data[alias])
	})
}

/* -------------------------------------------------------------------------- */
func TestInMemoryRepo_Get(t *testing.T) {
	repo := NewInMemoryRepo()
	ctx := context.Background()

	const alias = "exists"
	const original = "http://example.com"
	repo.data[alias] = original

	t.Run("get existing", func(t *testing.T) {
		res, err := repo.Get(ctx, alias)

		assert.NoError(t, err)
		assert.Equal(t, original, res)
	})

	t.Run("not found", func(t *testing.T) {
		res, err := repo.Get(ctx, "nonexistent")

		assert.Empty(t, res)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}
