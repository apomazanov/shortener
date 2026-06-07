package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepoConfig struct {
	file string
}

/* -------------------------------------------------------------------------- */
func (m *mockRepoConfig) GetStorageFile() string {
	return m.file
}

/* -------------------------------------------------------------------------- */
func TestFileRepo(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test_db.json")
	logger := zerolog.Nop()
	cfg := &mockRepoConfig{file: dbFile}

	t.Run("NewLocalStorage creates file", func(t *testing.T) {
		repo, err := NewLocalStorage(cfg, &logger)
		require.NoError(t, err)
		assert.NotNil(t, repo)

		_, err = os.Stat(dbFile)
		assert.NoError(t, err)
	})

	t.Run("Save and Get", func(t *testing.T) {
		repo, err := NewLocalStorage(cfg, &logger)
		require.NoError(t, err)

		alias := "test1"
		original := "http://example.com"
		ctx := context.Background()

		res, err := repo.Save(ctx, alias, original)
		assert.NoError(t, err)
		assert.Equal(t, alias, res)

		resGet, err := repo.Get(ctx, alias)
		assert.NoError(t, err)
		assert.Equal(t, original, resGet)
	})

	t.Run("Duplicate Alias", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "dup.json")
		repo, err := NewLocalStorage(&mockRepoConfig{file: testFile}, &logger)
		require.NoError(t, err)

		alias := "dup"
		original := "http://example.com"
		ctx := context.Background()

		_, err = repo.Save(ctx, alias, original)
		require.NoError(t, err)

		_, err = repo.Save(ctx, alias, "http://another.com")
		assert.ErrorIs(t, err, domain.ErrDuplicate)
	})

	t.Run("Existing Original URL", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "existing_url.json")
		repo, err := NewLocalStorage(&mockRepoConfig{file: testFile}, &logger)
		require.NoError(t, err)

		alias1 := "alias1"
		original := "http://example.com"
		ctx := context.Background()

		// First save
		res1, err := repo.Save(ctx, alias1, original)
		require.NoError(t, err)
		assert.Equal(t, alias1, res1)

		// Second save with same URL but different suggested alias
		alias2 := "alias2"
		res2, err := repo.Save(ctx, alias2, original)
		require.NoError(t, err)
		// Should return the first alias instead of error or new entry
		assert.Equal(t, alias1, res2)
	})

	t.Run("Persistence", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "persist.json")
		alias := "persist"
		original := "http://persistent.com"
		ctx := context.Background()

		// First instance: Save data
		repo1, err := NewLocalStorage(&mockRepoConfig{file: testFile}, &logger)
		require.NoError(t, err)
		_, err = repo1.Save(ctx, alias, original)
		require.NoError(t, err)

		// Second instance: Load data from the same file
		repo2, err := NewLocalStorage(&mockRepoConfig{file: testFile}, &logger)
		require.NoError(t, err)

		resGet, err := repo2.Get(ctx, alias)
		assert.NoError(t, err)
		assert.Equal(t, original, resGet)

		// Verify cache was loaded correctly
		cached, exists := repo2.cache[alias]
		assert.True(t, exists)
		assert.Equal(t, original, cached)
	})

	t.Run("UUID generation and increment", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "uuid.json")
		repo, err := NewLocalStorage(&mockRepoConfig{file: testFile}, &logger)
		require.NoError(t, err)
		ctx := context.Background()

		// Save first entry
		_, err = repo.Save(ctx, "alias1", "http://url1.com")
		require.NoError(t, err)
		assert.Equal(t, 1, repo.lastUUID, "lastUUID should be 1 after first save")

		// Save second entry
		_, err = repo.Save(ctx, "alias2", "http://url2.com")
		require.NoError(t, err)
		assert.Equal(t, 2, repo.lastUUID, "lastUUID should be 2 after second save")

		// Verify persistence of UUIDs by reloading
		// This is covered by the Persistence test, but good to keep in mind.
	})

	t.Run("Get Not Found", func(t *testing.T) {
		repo, err := NewLocalStorage(cfg, &logger)
		require.NoError(t, err)

		res, err := repo.Get(context.Background(), "missing")
		assert.Empty(t, res)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("NewLocalStorage creates nested directories", func(t *testing.T) {
		// проверяем, что NewLocalStorage создает вложенные директории, если они отсутствуют
		nestedFile := filepath.Join(t.TempDir(), "a", "b", "c", "db.json")
		nestedCfg := &mockRepoConfig{file: nestedFile}

		repo, err := NewLocalStorage(nestedCfg, &logger)
		require.NoError(t, err)
		assert.NotNil(t, repo)
		assert.FileExists(t, nestedFile)
	})
}
