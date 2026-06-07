package repository

import (
	"context"
	"testing"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestMemStorage(t *testing.T) {
	log := zerolog.Nop()
	repo := NewMemStorage(&log)
	ctx := context.Background()

	t.Run("Save and Get", func(t *testing.T) {
		alias := "test1"
		original := "http://example1.com"

		res, err := repo.Save(ctx, alias, original)
		assert.NoError(t, err)
		assert.Equal(t, alias, res)

		resGet, err := repo.Get(ctx, alias)
		assert.NoError(t, err)
		assert.Equal(t, original, resGet)
	})

	t.Run("Duplicate Alias", func(t *testing.T) {
		alias := "dup"

		_, err := repo.Save(ctx, alias, "http://example2.com")
		assert.NoError(t, err)

		_, err = repo.Save(ctx, alias, "http://another.com")
		assert.ErrorIs(t, err, domain.ErrAliasDuplicate)
	})

	t.Run("Existing Original URL", func(t *testing.T) {
		repo := NewMemStorage(&log)
		alias1 := "alias1"
		original := "http://example3.com"

		res1, err := repo.Save(ctx, alias1, original)
		assert.NoError(t, err)
		assert.Equal(t, alias1, res1)

		alias2 := "alias2"
		res2, err := repo.Save(ctx, alias2, original)
		assert.ErrorIs(t, err, domain.ErrOriginalURLDuplicate)
		// Должен вернуть существующий alias1
		assert.Equal(t, alias1, res2)
	})

	t.Run("Get Not Found", func(t *testing.T) {
		res, err := repo.Get(ctx, "missing")
		assert.Empty(t, res)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("Aborted by Context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := repo.Save(ctx, "a", "b")
		assert.ErrorContains(t, err, "repo: save aborted")
	})
}
