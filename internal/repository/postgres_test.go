package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresStorage_Save(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &PostgresStorage{pool: mock}
	alias := "test_alias"
	original := "http://example.com"

	t.Run("success insert", func(t *testing.T) {
		mock.ExpectQuery("SELECT alias FROM urls").
			WithArgs(original).
			WillReturnError(pgx.ErrNoRows)

		mock.ExpectExec("INSERT INTO urls").
			WithArgs(alias, original).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		res, err := repo.Save(context.Background(), alias, original)
		assert.NoError(t, err)
		assert.Equal(t, alias, res)
	})

	t.Run("success existing", func(t *testing.T) {
		existingAlias := "old_alias"
		mock.ExpectQuery("SELECT alias FROM urls").
			WithArgs(original).
			WillReturnRows(pgxmock.NewRows([]string{"alias"}).AddRow(existingAlias))

		res, err := repo.Save(context.Background(), alias, original)
		assert.NoError(t, err)
		assert.Equal(t, existingAlias, res)
	})

	t.Run("duplicate alias", func(t *testing.T) {
		mock.ExpectQuery("SELECT alias FROM urls").
			WithArgs(original).
			WillReturnError(pgx.ErrNoRows)

		mock.ExpectExec("INSERT INTO urls").
			WithArgs(alias, original).
			WillReturnResult(pgxmock.NewResult("INSERT", 0))

		_, err := repo.Save(context.Background(), alias, original)
		assert.ErrorIs(t, err, domain.ErrDuplicate)
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT alias FROM urls").
			WillReturnError(errors.New("db fail"))

		_, err := repo.Save(context.Background(), alias, original)
		assert.ErrorContains(t, err, "repo: database error")
	})

	t.Run("aborted by context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Отменяем сразу

		_, err := repo.Save(ctx, alias, original)
		assert.ErrorContains(t, err, "repo: save aborted")
	})
}

func TestPostgresStorage_Get(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &PostgresStorage{pool: mock}
	alias := "test_alias"
	original := "http://example.com"

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery("SELECT original FROM urls").
			WithArgs(alias).
			WillReturnRows(pgxmock.NewRows([]string{"original"}).AddRow(original))

		res, err := repo.Get(context.Background(), alias)
		assert.NoError(t, err)
		assert.Equal(t, original, res)
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT original FROM urls").
			WithArgs(alias).
			WillReturnError(pgx.ErrNoRows)

		res, err := repo.Get(context.Background(), alias)
		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.Empty(t, res)
	})

	t.Run("query error", func(t *testing.T) {
		mock.ExpectQuery("SELECT original FROM urls").
			WillReturnError(errors.New("scan error"))

		res, err := repo.Get(context.Background(), alias)
		assert.ErrorContains(t, err, "database error")
		assert.Empty(t, res)
	})
}

func TestPostgresStorage_Ping(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &PostgresStorage{pool: mock}

	t.Run("ping success", func(t *testing.T) {
		mock.ExpectPing()
		assert.NoError(t, repo.Ping(context.Background()))
	})

	t.Run("ping failure", func(t *testing.T) {
		mock.ExpectPing().WillReturnError(errors.New("down"))
		assert.Error(t, repo.Ping(context.Background()))
	})
}
