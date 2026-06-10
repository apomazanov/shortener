package service

import (
	"context"
	"errors"
	"testing"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/stretchr/testify/assert"
)

/* ------------------------------- Repo mock -------------------------------- */
type repoMock struct {
	original  string
	batch     map[string]string
	err       error
	saveCalls int
}

func (r *repoMock) Save(ctx context.Context, alias string, original string) (string, error) {
	r.saveCalls++
	return alias, r.err
}

func (r *repoMock) SaveBatch(ctx context.Context, toWrite map[string]string) (map[string]string, error) {
	r.saveCalls++
	return r.batch, r.err
}

func (r *repoMock) Get(ctx context.Context, alias string) (string, error) {
	return r.original, r.err
}

/* -------------------------------------------------------------------------- */
func TestGetOriginalURL(t *testing.T) {
	repo := &repoMock{}
	s := New(repo)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo.err = nil
		repo.original = "http://example.com"

		url, err := s.GetOriginalURL(ctx, "alias1")

		assert.NoError(t, err)
		assert.Equal(t, "http://example.com", url)
	})

	t.Run("not found", func(t *testing.T) {
		repo.err = domain.ErrNotFound
		repo.original = ""

		url, err := s.GetOriginalURL(ctx, "unknown")

		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.Empty(t, url)
	})

	t.Run("internal error", func(t *testing.T) {
		internalErr := errors.New("db failure")
		repo.err = internalErr
		repo.original = ""

		url, err := s.GetOriginalURL(ctx, "alias1")

		assert.ErrorIs(t, err, internalErr)
		assert.Empty(t, url)
	})
}

/* -------------------------------------------------------------------------- */
func TestCreateURLAlias(t *testing.T) {
	repo := &repoMock{}
	s := New(repo)
	ctx := context.Background()

	t.Run("success first try", func(t *testing.T) {
		repo.err = nil
		repo.saveCalls = 0

		alias, err := s.CreateURLAlias(ctx, "http://google.com")

		assert.NoError(t, err)
		assert.Len(t, alias, 6)
		assert.Equal(t, 1, repo.saveCalls)
	})

	t.Run("repo error", func(t *testing.T) {
		someErr := errors.New("disk full")
		repo.err = someErr
		repo.saveCalls = 0

		alias, err := s.CreateURLAlias(ctx, "http://google.com")

		assert.ErrorIs(t, err, someErr)
		assert.Empty(t, alias)
		assert.Equal(t, 1, repo.saveCalls)
	})

	t.Run("retry limit exceeded on duplicates", func(t *testing.T) {
		repo.err = domain.ErrAliasDuplicate
		repo.saveCalls = 0

		alias, err := s.CreateURLAlias(ctx, "http://google.com")

		assert.ErrorIs(t, err, domain.ErrSaveRetryLimitExceeded)
		assert.Empty(t, alias)
		assert.Equal(t, 5, repo.saveCalls)
	})
}

/* -------------------------------------------------------------------------- */
func TestCreateURLAliasBatch(t *testing.T) {
	repo := &repoMock{}
	s := New(repo)
	ctx := context.Background()
	urls := []string{"http://url1.com", "http://url2.com"}

	t.Run("success", func(t *testing.T) {
		repo.err = nil
		repo.saveCalls = 0
		repo.batch = map[string]string{
			"http://url1.com": "a1",
			"http://url2.com": "a2",
		}

		res, err := s.CreateURLAliasBatch(ctx, urls)

		assert.NoError(t, err)
		assert.Len(t, res, 2)
		assert.Equal(t, 1, repo.saveCalls)
	})

	t.Run("retry on alias collision", func(t *testing.T) {
		repo.saveCalls = 0
		// First call fails with alias duplicate, second succeeds
		repo.err = domain.ErrAliasDuplicate

		// Note: The current service implementation retries the full batch.
		// After 5 attempts it will return error if we don't change repo.err.
		_, err := s.CreateURLAliasBatch(ctx, urls)
		assert.ErrorIs(t, err, domain.ErrSaveRetryLimitExceeded)
		assert.Equal(t, 5, repo.saveCalls)
	})

	t.Run("original conflict", func(t *testing.T) {
		repo.err = domain.ErrOriginalURLDuplicate
		repo.saveCalls = 0
		repo.batch = map[string]string{"http://url1.com": "existing"}

		res, err := s.CreateURLAliasBatch(ctx, urls)

		assert.ErrorIs(t, err, domain.ErrOriginalURLDuplicate)
		assert.NotNil(t, res)
	})
}
