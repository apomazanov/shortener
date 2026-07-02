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

type postgresTestContext struct {
	mock pgxmock.PgxPoolIface
	repo *PostgresStorage
}

func createPostgresTestContext(t *testing.T) *postgresTestContext {
	t.Helper()

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)

	t.Cleanup(func() {
		mock.Close()
	})

	return &postgresTestContext{
		mock: mock,
		repo: &PostgresStorage{pool: mock},
	}
}

func TestPostgresStorage_Save(t *testing.T) {
	alias := "test_alias"
	original := "http://example.com"
	userID := "user-1"

	t.Run("success insert", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		tcx.mock.ExpectQuery("(?s)INSERT INTO urls .* RETURNING alias").
			WithArgs(alias, original, userID).
			WillReturnRows(pgxmock.NewRows([]string{"alias"}).AddRow(alias))

		res, err := tcx.repo.Save(context.Background(), alias, original, userID)

		require.NoError(t, err)
		assert.Equal(t, alias, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})

	t.Run("success existing", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		existingAlias := "old_alias"

		tcx.mock.ExpectQuery("(?s)INSERT INTO urls .* RETURNING alias").
			WithArgs(alias, original, userID).
			WillReturnRows(pgxmock.NewRows([]string{"alias"}).AddRow(existingAlias))

		res, err := tcx.repo.Save(context.Background(), alias, original, userID)

		require.ErrorIs(t, err, domain.ErrOriginalURLDuplicate)
		assert.Equal(t, existingAlias, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})

	t.Run("duplicate alias", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		tcx.mock.ExpectQuery("(?s)INSERT INTO urls .* RETURNING alias").
			WithArgs(alias, original, userID).
			WillReturnError(&pgconn.PgError{
				Code:           pgerrcode.UniqueViolation,
				ConstraintName: constraintUniqueAlias,
			})

		res, err := tcx.repo.Save(context.Background(), alias, original, userID)

		require.ErrorIs(t, err, domain.ErrAliasDuplicate)
		assert.Empty(t, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		tcx.mock.ExpectQuery("(?s)INSERT INTO urls .* RETURNING alias").
			WithArgs(alias, original, userID).
			WillReturnError(errors.New("db fail"))

		res, err := tcx.repo.Save(context.Background(), alias, original, userID)

		require.ErrorContains(t, err, "repo: database error")
		assert.Empty(t, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})

	t.Run("aborted by context", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		res, err := tcx.repo.Save(ctx, alias, original, userID)

		require.ErrorContains(t, err, "repo: save aborted")
		assert.Empty(t, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})
}

func TestPostgresStorage_SaveBatch(t *testing.T) {
	batch := map[string]string{
		"http://u1.com": "a1",
		"http://u2.com": "a2",
	}

	userID := "user-1"
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		tcx.mock.ExpectQuery("(?s)INSERT INTO urls .* RETURNING alias, original").
			WithArgs(
				pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
				pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			).
			WillReturnRows(pgxmock.NewRows([]string{"alias", "original"}).
				AddRow("a1", "http://u1.com").
				AddRow("a2", "http://u2.com"))

		res, err := tcx.repo.SaveBatch(ctx, batch, userID)

		require.NoError(t, err)
		assert.Len(t, res, 2)
		assert.Equal(t, "a1", res["http://u1.com"])
		assert.Equal(t, "a2", res["http://u2.com"])
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})

	t.Run("alias conflict", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		tcx.mock.ExpectQuery("(?s)INSERT INTO urls .* RETURNING alias, original").
			WithArgs(
				pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
				pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			).
			WillReturnError(&pgconn.PgError{
				Code:           pgerrcode.UniqueViolation,
				ConstraintName: constraintUniqueAlias,
			})

		res, err := tcx.repo.SaveBatch(ctx, batch, userID)

		require.ErrorIs(t, err, domain.ErrAliasDuplicate)
		assert.Nil(t, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})

	t.Run("original conflict", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		tcx.mock.ExpectQuery("(?s)INSERT INTO urls .* RETURNING alias, original").
			WithArgs(
				pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
				pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			).
			WillReturnRows(pgxmock.NewRows([]string{"alias", "original"}).
				AddRow("existing_alias", "http://u1.com").
				AddRow("a2", "http://u2.com"))

		res, err := tcx.repo.SaveBatch(ctx, batch, userID)

		require.ErrorIs(t, err, domain.ErrOriginalURLDuplicate)
		assert.NotNil(t, res)
		assert.Equal(t, "existing_alias", res["http://u1.com"])
		assert.Equal(t, "a2", res["http://u2.com"])
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		tcx.mock.ExpectQuery("(?s)INSERT INTO urls .* RETURNING alias, original").
			WithArgs(
				pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
				pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			).
			WillReturnError(errors.New("db fail"))

		res, err := tcx.repo.SaveBatch(ctx, batch, userID)

		require.ErrorContains(t, err, "repo: batch database error")
		assert.Nil(t, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})

	t.Run("empty batch", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		res, err := tcx.repo.SaveBatch(ctx, nil, userID)

		require.NoError(t, err)
		assert.Nil(t, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})

	t.Run("aborted by context", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		res, err := tcx.repo.SaveBatch(ctx, batch, userID)

		require.ErrorContains(t, err, "repo: batch save aborted")
		assert.Nil(t, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})
}

func TestPostgresStorage_Get(t *testing.T) {
	alias := "test_alias"
	original := "http://example.com"

	t.Run("success", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		tcx.mock.ExpectQuery("(?s)SELECT original, is_deleted\\s+FROM urls\\s+WHERE alias = \\$1").
			WithArgs(alias).
			WillReturnRows(pgxmock.NewRows([]string{"original", "is_deleted"}).AddRow(original, false))

		res, err := tcx.repo.Get(context.Background(), alias)

		require.NoError(t, err)
		assert.Equal(t, original, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})

	t.Run("deleted", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		tcx.mock.ExpectQuery("(?s)SELECT original, is_deleted\\s+FROM urls\\s+WHERE alias = \\$1").
			WithArgs(alias).
			WillReturnRows(pgxmock.NewRows([]string{"original", "is_deleted"}).AddRow(original, true))

		res, err := tcx.repo.Get(context.Background(), alias)

		require.ErrorIs(t, err, domain.ErrFoundDeleted)
		assert.Empty(t, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		tcx.mock.ExpectQuery("(?s)SELECT original, is_deleted\\s+FROM urls\\s+WHERE alias = \\$1").
			WithArgs(alias).
			WillReturnError(pgx.ErrNoRows)

		res, err := tcx.repo.Get(context.Background(), alias)

		require.ErrorIs(t, err, domain.ErrNotFound)
		assert.Empty(t, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		tcx.mock.ExpectQuery("(?s)SELECT original, is_deleted\\s+FROM urls\\s+WHERE alias = \\$1").
			WithArgs(alias).
			WillReturnError(errors.New("scan error"))

		res, err := tcx.repo.Get(context.Background(), alias)

		require.ErrorContains(t, err, "repo: database error")
		assert.Empty(t, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})

	t.Run("aborted by context", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		res, err := tcx.repo.Get(ctx, alias)

		require.ErrorContains(t, err, "repo: get aborted")
		assert.Empty(t, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})
}

func TestPostgresStorage_GetByUser(t *testing.T) {
	userID := "user-1"

	t.Run("success", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		tcx.mock.ExpectQuery("(?s)SELECT alias, original, is_deleted\\s+FROM urls\\s+WHERE user_id = \\$1").
			WithArgs(userID).
			WillReturnRows(pgxmock.NewRows([]string{"alias", "original", "is_deleted"}).
				AddRow("a1", "http://u1.com", false).
				AddRow("a2", "http://u2.com", false))

		res, err := tcx.repo.GetByUser(context.Background(), userID)

		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			"a1": "http://u1.com",
			"a2": "http://u2.com",
		}, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})

	t.Run("empty result", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		tcx.mock.ExpectQuery("(?s)SELECT alias, original, is_deleted\\s+FROM urls\\s+WHERE user_id = \\$1").
			WithArgs(userID).
			WillReturnRows(pgxmock.NewRows([]string{"alias", "original", "is_deleted"}))

		res, err := tcx.repo.GetByUser(context.Background(), userID)

		require.NoError(t, err)
		assert.Empty(t, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})

	t.Run("deleted items are filtered", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		tcx.mock.ExpectQuery("(?s)SELECT alias, original, is_deleted\\s+FROM urls\\s+WHERE user_id = \\$1").
			WithArgs(userID).
			WillReturnRows(pgxmock.NewRows([]string{"alias", "original", "is_deleted"}).
				AddRow("a1", "http://u1.com", false).
				AddRow("a2", "http://u2.com", true). // This one should be filtered out
				AddRow("a3", "http://u3.com", false))

		res, err := tcx.repo.GetByUser(context.Background(), userID)

		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			"a1": "http://u1.com",
			"a3": "http://u3.com",
		}, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		tcx.mock.ExpectQuery("(?s)SELECT alias, original, is_deleted\\s+FROM urls\\s+WHERE user_id = \\$1").
			WithArgs(userID).
			WillReturnError(errors.New("db fail"))

		res, err := tcx.repo.GetByUser(context.Background(), userID)

		require.ErrorContains(t, err, "repo: database error")
		assert.Nil(t, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})

	t.Run("rows error", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		tcx.mock.ExpectQuery("(?s)SELECT alias, original, is_deleted\\s+FROM urls\\s+WHERE user_id = \\$1").
			WithArgs(userID).
			WillReturnRows(pgxmock.NewRows([]string{"alias", "original"}).
				AddRow("a1", "http://u1.com").
				RowError(0, errors.New("rows error")))

		res, err := tcx.repo.GetByUser(context.Background(), userID)

		require.ErrorContains(t, err, "repo: database error")
		assert.Nil(t, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})

	t.Run("aborted by context", func(t *testing.T) {
		tcx := createPostgresTestContext(t)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		res, err := tcx.repo.GetByUser(ctx, userID)

		require.ErrorContains(t, err, "repo: get aborted")
		assert.Nil(t, res)
		assert.NoError(t, tcx.mock.ExpectationsWereMet())
	})
}
