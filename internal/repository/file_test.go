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
func (m *mockRepoConfig) GetRepoFile() string {
	return m.file
}

/* -------------------------------------------------------------------------- */
func TestFileRepo(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test_db.json")
	logger := zerolog.Nop()
	cfg := &mockRepoConfig{file: dbFile}

	t.Run("NewFileRepo creates file", func(t *testing.T) {
		repo, err := NewFileRepo(cfg, &logger)
		require.NoError(t, err)
		assert.NotNil(t, repo)

		_, err = os.Stat(dbFile)
		assert.NoError(t, err)
	})

	t.Run("Save and Get", func(t *testing.T) {
		repo, err := NewFileRepo(cfg, &logger)
		require.NoError(t, err)

		alias := "test1"
		original := "http://example.com"
		ctx := context.Background()

		err = repo.Save(ctx, alias, original)
		assert.NoError(t, err)

		res, err := repo.Get(ctx, alias)
		assert.NoError(t, err)
		assert.Equal(t, original, res)
	})

	t.Run("Duplicate Alias", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "dup.json")
		repo, err := NewFileRepo(&mockRepoConfig{file: testFile}, &logger)
		require.NoError(t, err)

		alias := "dup"
		original := "http://example.com"
		ctx := context.Background()

		err = repo.Save(ctx, alias, original)
		require.NoError(t, err)

		err = repo.Save(ctx, alias, "http://another.com")
		assert.ErrorIs(t, err, domain.ErrDuplicate)
	})

	t.Run("Persistence", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "persist.json")
		alias := "persist"
		original := "http://persistent.com"
		ctx := context.Background()

		// First instance: Save data
		repo1, err := NewFileRepo(&mockRepoConfig{file: testFile}, &logger)
		require.NoError(t, err)
		err = repo1.Save(ctx, alias, original)
		require.NoError(t, err)

		// Second instance: Load data from the same file
		repo2, err := NewFileRepo(&mockRepoConfig{file: testFile}, &logger)
		require.NoError(t, err)

		res, err := repo2.Get(ctx, alias)
		assert.NoError(t, err)
		assert.Equal(t, original, res)

		// Verify cache was loaded correctly
		cached, exists := repo2.cache[alias]
		assert.True(t, exists)
		assert.Equal(t, original, cached)
	})

	t.Run("Get Not Found", func(t *testing.T) {
		repo, err := NewFileRepo(cfg, &logger)
		require.NoError(t, err)

		res, err := repo.Get(context.Background(), "missing")
		assert.Empty(t, res)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("NewFileRepo Invalid Path", func(t *testing.T) {
		// directory /nonexistent_path_test does not exist
		badCfg := &mockRepoConfig{file: "/nonexistent_path_test/db.json"}
		repo, err := NewFileRepo(badCfg, &logger)

		assert.Error(t, err)
		assert.Nil(t, repo)
	})
}