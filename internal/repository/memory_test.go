package repository

import (
	"context"
	"testing"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemStorage(t *testing.T) {
	repo := NewMemStorage()
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

	t.Run("SaveBatch success", func(t *testing.T) {
		batch := map[string]string{
			"http://batch1.com": "b1",
			"http://batch2.com": "b2",
		}

		written, err := repo.SaveBatch(ctx, batch)
		assert.NoError(t, err)
		assert.Len(t, written, 2)
		assert.Equal(t, "b1", written["http://batch1.com"])
		assert.Equal(t, "b2", written["http://batch2.com"])

		// Verify cache
		val, _ := repo.Get(ctx, "b1")
		assert.Equal(t, "http://batch1.com", val)
		val, _ = repo.Get(ctx, "b2")
		assert.Equal(t, "http://batch2.com", val)
	})

	t.Run("SaveBatch with existing alias in cache", func(t *testing.T) {
		_, err := repo.Save(ctx, "existingAlias", "http://existing.com")
		require.NoError(t, err)

		batch := map[string]string{
			"http://new1.com": "new1",
			"http://new2.com": "existingAlias", // Conflict with existing
		}

		written, err := repo.SaveBatch(ctx, batch)
		assert.ErrorIs(t, err, domain.ErrAliasDuplicate)
		assert.Nil(t, written) // No partial write

		// Verify cache is unchanged
		_, err = repo.Get(ctx, "new1")
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("SaveBatch with intra-batch alias collision", func(t *testing.T) {
		batch := map[string]string{
			"http://url1.com": "sameAlias",
			"http://url2.com": "sameAlias", // Collision within batch
		}

		written, err := repo.SaveBatch(ctx, batch)
		assert.ErrorIs(t, err, domain.ErrAliasDuplicate)
		assert.Nil(t, written) // No partial write

		// Verify cache is unchanged
		_, err = repo.Get(ctx, "sameAlias")
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("SaveBatch with existing original URL", func(t *testing.T) {
		_, err := repo.Save(ctx, "firstAlias", "http://original.com")
		require.NoError(t, err)

		batch := map[string]string{
			"http://new.com":      "newAlias",
			"http://original.com": "anotherAlias", // Conflict with existing original
		}

		written, err := repo.SaveBatch(ctx, batch)
		assert.ErrorIs(t, err, domain.ErrOriginalURLDuplicate)
		assert.NotNil(t, written) // Partial success for non-conflicting items
		assert.Len(t, written, 2)
		assert.Equal(t, "newAlias", written["http://new.com"])
		assert.Equal(t, "firstAlias", written["http://original.com"]) // Should return existing alias

		// Verify cache is updated for newAlias, but not for anotherAlias
		val, _ := repo.Get(ctx, "newAlias")
		assert.Equal(t, "http://new.com", val)
		val, _ = repo.Get(ctx, "firstAlias")
		assert.Equal(t, "http://original.com", val)
		_, err = repo.Get(ctx, "anotherAlias") // Should not be in cache
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("SaveBatch aborted by context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		batch := map[string]string{
			"http://test.com": "t1",
		}

		written, err := repo.SaveBatch(ctx, batch)
		assert.ErrorContains(t, err, "repo: batch save aborted")
		assert.Nil(t, written)
	})
}
