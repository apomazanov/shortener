package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
		mock.ExpectQuery("INSERT INTO urls").
			WithArgs(alias, original).
			WillReturnRows(pgxmock.NewRows([]string{"alias"}).AddRow(alias))

		res, err := repo.Save(context.Background(), alias, original)
		assert.NoError(t, err)
		assert.Equal(t, alias, res)
	})

	t.Run("success existing", func(t *testing.T) {
		existingAlias := "old_alias"
		mock.ExpectQuery("INSERT INTO urls").
			WithArgs(alias, original).
			WillReturnRows(pgxmock.NewRows([]string{"alias"}).AddRow(existingAlias))

		res, err := repo.Save(context.Background(), alias, original)
		assert.ErrorIs(t, err, domain.ErrOriginalURLDuplicate)
		assert.Equal(t, existingAlias, res)
	})

	t.Run("duplicate alias", func(t *testing.T) {
		mock.ExpectQuery("INSERT INTO urls").
			WithArgs(alias, original).
			WillReturnError(&pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: constraintUniqueAlias})

		_, err := repo.Save(context.Background(), alias, original)
		assert.ErrorIs(t, err, domain.ErrAliasDuplicate)
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("INSERT INTO urls").
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

func TestPostgresStorage_SaveBatch(t *testing.T) {
	batch := map[string]string{"http://u1.com": "a1", "http://u2.com": "a2"}
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := &PostgresStorage{pool: mock}

		mock.ExpectQuery("^INSERT INTO urls .*").
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"alias", "original"}).
				AddRow("a1", "http://u1.com").
				AddRow("a2", "http://u2.com"))

		res, err := repo.SaveBatch(ctx, batch)
		require.NoError(t, err)
		assert.Len(t, res, 2)
		assert.Equal(t, "a1", res["http://u1.com"])
		assert.Equal(t, "a2", res["http://u2.com"])
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("alias conflict", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := &PostgresStorage{pool: mock}

		mock.ExpectQuery("^INSERT INTO urls .*").
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(&pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: constraintUniqueAlias})

		_, err = repo.SaveBatch(ctx, batch)
		assert.ErrorIs(t, err, domain.ErrAliasDuplicate)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("original conflict", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := &PostgresStorage{pool: mock}

		// DB returns existing alias for one of the URLs
		mock.ExpectQuery("^INSERT INTO urls .*").
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"alias", "original"}).
				AddRow("existing_alias", "http://u1.com").
				AddRow("a2", "http://u2.com"))

		res, err := repo.SaveBatch(ctx, batch)
		require.ErrorIs(t, err, domain.ErrOriginalURLDuplicate)
		assert.NotNil(t, res)
		assert.Equal(t, "existing_alias", res["http://u1.com"])
		assert.Equal(t, "a2", res["http://u2.com"])
		assert.NoError(t, mock.ExpectationsWereMet())
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
