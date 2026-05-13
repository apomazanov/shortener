package service

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

/* ------------------------------- Repo mock -------------------------------- */
type repoMock struct {
	original  string
	err       error
	saveCalls int
}

func (r *repoMock) Save(ctx context.Context, alias string, original string) error {
	r.saveCalls++
	return r.err
}

func (r *repoMock) Get(ctx context.Context, alias string) (string, error) {
	return r.original, r.err
}

/* -------------------------------------------------------------------------- */
func TestGetOriginalUrl(t *testing.T) {
	repo := &repoMock{}
	logger := zerolog.New(io.Discard)
	s := New(repo, &logger)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo.err = nil
		repo.original = "http://example.com"

		url, err := s.GetOriginalUrl(ctx, "alias1")

		assert.NoError(t, err)
		assert.Equal(t, "http://example.com", url)
	})

	t.Run("not found", func(t *testing.T) {
		repo.err = domain.ErrNotFound
		repo.original = ""

		url, err := s.GetOriginalUrl(ctx, "unknown")

		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.Empty(t, url)
	})

	t.Run("internal error", func(t *testing.T) {
		internalErr := errors.New("db failure")
		repo.err = internalErr
		repo.original = ""

		url, err := s.GetOriginalUrl(ctx, "alias1")

		assert.ErrorIs(t, err, internalErr)
		assert.Empty(t, url)
	})
}

/* -------------------------------------------------------------------------- */
func TestCreateUrlAlias(t *testing.T) {
	repo := &repoMock{}
	logger := zerolog.New(io.Discard)
	s := New(repo, &logger)
	ctx := context.Background()

	t.Run("success first try", func(t *testing.T) {
		repo.err = nil
		repo.saveCalls = 0

		alias, err := s.CreateUrlAlias(ctx, "http://google.com")

		assert.NoError(t, err)
		assert.Len(t, alias, 6)
		assert.Equal(t, 1, repo.saveCalls)
	})

	t.Run("repo error", func(t *testing.T) {
		someErr := errors.New("disk full")
		repo.err = someErr
		repo.saveCalls = 0

		alias, err := s.CreateUrlAlias(ctx, "http://google.com")

		assert.ErrorIs(t, err, someErr)
		assert.Empty(t, alias)
		assert.Equal(t, 1, repo.saveCalls)
	})

	t.Run("retry limit exceeded on duplicates", func(t *testing.T) {
		repo.err = domain.ErrDuplicate
		repo.saveCalls = 0

		alias, err := s.CreateUrlAlias(ctx, "http://google.com")

		assert.ErrorIs(t, err, domain.ErrSaveRetryLimitExceeded)
		assert.Empty(t, alias)
		assert.Equal(t, 5, repo.saveCalls)
	})
}
