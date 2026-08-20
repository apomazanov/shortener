package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/apomazanov/shortener/internal/domain"
	"github.com/apomazanov/shortener/migrations"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	// constraintUniqueAlias is a name of unique alias constraint in postgre.
	constraintUniqueAlias = "idx_urls_unique_alias"
	// constraintUniqueOriginal is a name of unique origin constraint in postgre.
	constraintUniqueOriginal = "idx_urls_unique_original"
)

// pgxPooler contains database methods used by this package.
type pgxPooler interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Ping(ctx context.Context) error
	Close()
}

// PostgresConfig defines configurations methods for database operations.
type PostgresConfig interface {
	GetDatabaseDSN() string
	GetNoDBMigration() bool
}

// PostgresStorage contains data for database operations.
type PostgresStorage struct {
	// pool is a pool of database connections used for data transfer.
	pool pgxPooler
}

// runMigrations starts a migration routine from 'migrations' directory.
func runMigrations(ctx context.Context, dsn string) error {

	goose.SetBaseFS(migrations.EmbedFS)

	err := goose.SetDialect("postgres")
	if err != nil {
		return fmt.Errorf("migration: cannot set dialect: %w", err)
	}

	db, err := goose.OpenDBWithDriver("pgx", dsn)
	if err != nil {
		return fmt.Errorf("migration: cannot open DB: %w", err)
	}
	defer db.Close()

	err = goose.UpContext(ctx, db, ".")
	if err != nil {
		return fmt.Errorf("migration: cannot apply migrations: %w", err)
	}

	return nil
}

// NewPostgresStorage creates a new PostgreSQL storage.
func NewPostgresStorage(cfg PostgresConfig) (*PostgresStorage, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dsn := cfg.GetDatabaseDSN()
	doNotMigrate := cfg.GetNoDBMigration()

	// DB migrations

	if !doNotMigrate {
		err := runMigrations(ctx, dsn)
		if err != nil {
			return nil, fmt.Errorf("repo: db migration failed: %w", err)
		}
	}

	// Connections pool for business

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("repo: cannot parse DSN: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("repo: cannot create connections pool: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("repo: cannot ping database: %w", err)
	}

	return &PostgresStorage{pool: pool}, nil
}

// Save provides write operation of single recoed to postgre storage.
func (r *PostgresStorage) Save(
	ctx context.Context,
	alias string,
	original string,
	userID string,
) (writtenAlias string, err error) {

	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("repo: save aborted: %w", err)
	}

	queryCtx := context.WithoutCancel(ctx)
	queryCtx, cancel := context.WithTimeout(queryCtx, 3*time.Second)
	defer cancel()

	// creating query

	query := `
		INSERT INTO urls (alias, original, user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (original) WHERE is_deleted = false
		DO UPDATE SET original = EXCLUDED.original
		RETURNING alias;
	`

	// executing

	err = r.pool.QueryRow(queryCtx, query, alias, original, userID).Scan(&writtenAlias)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			if pgErr.ConstraintName == constraintUniqueAlias {
				return "", domain.ErrAliasDuplicate
			}
		}

		return "", fmt.Errorf("repo: database error: %w", err)
	}

	if alias != writtenAlias {
		return writtenAlias, domain.ErrOriginalURLDuplicate
	}

	return writtenAlias, nil
}

// SaveBatch provides write operation of batch of records to postgre storage.
func (r *PostgresStorage) SaveBatch(
	ctx context.Context,
	toWrite map[string]string,
	userID string,
) (written map[string]string, err error) {

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("repo: batch save aborted: %w", err)
	}

	if len(toWrite) == 0 {
		return nil, nil
	}

	queryCtx := context.WithoutCancel(ctx)
	queryCtx, cancel := context.WithTimeout(queryCtx, 3*time.Second)
	defer cancel()

	// creating query

	builder := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		Insert("urls").
		Columns("alias", "original", "user_id")

	for original, alias := range toWrite {
		builder = builder.Values(alias, original, userID)
	}

	builder = builder.Suffix("ON CONFLICT (original) WHERE is_deleted = false DO UPDATE SET original = EXCLUDED.original RETURNING alias, original")

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("repo: cannot build batch query: %w", err)
	}

	// executing

	rows, err := r.pool.Query(queryCtx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			if pgErr.ConstraintName == constraintUniqueAlias {
				// dropping full batch
				return nil, domain.ErrAliasDuplicate
			}
		}

		return nil, fmt.Errorf("repo: batch database error: %w", err)
	}
	defer rows.Close()

	// reading data and parsing

	written = make(map[string]string, len(toWrite))
	isOriginalConflict := false

	for rows.Next() {
		var alias, original string

		err = rows.Scan(&alias, &original)
		if err != nil {
			return nil, fmt.Errorf("repo: batch cannot scan row: %w", err)
		}

		written[original] = alias

		if alias != toWrite[original] {
			isOriginalConflict = true
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repo: batch database error: %w", err)
	}

	if isOriginalConflict {
		return written, domain.ErrOriginalURLDuplicate
	}

	return written, nil

}

// Get provides a read operation of single record from postgre storage.
func (r *PostgresStorage) Get(ctx context.Context, alias string) (original string, err error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("repo: get aborted: %w", err)
	}

	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `
		SELECT original, is_deleted
		FROM urls
		WHERE alias = $1;
	`
	var isDeleted bool

	err = r.pool.QueryRow(queryCtx, query, alias).Scan(&original, &isDeleted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", domain.ErrNotFound
		}
		return "", fmt.Errorf("repo: database error: %w", err)
	}

	if isDeleted {
		return "", domain.ErrFoundDeleted
	}

	return original, nil
}

// GetByUser returns a list of aliases and original URLs stored by certain user.
func (r *PostgresStorage) GetByUser(ctx context.Context, userID string) (data map[string]string, err error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("repo: get aborted: %w", err)
	}

	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `
		SELECT alias, original, is_deleted
		FROM urls
		WHERE user_id = $1;
	`

	rows, err := r.pool.Query(queryCtx, query, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("repo: database error: %w", err)
	}
	defer rows.Close()

	data = make(map[string]string)
	for rows.Next() {
		var k, v string
		var isDeleted bool

		err := rows.Scan(&k, &v, &isDeleted)
		if err != nil {
			return nil, fmt.Errorf("repo: database error: %w", err)
		}

		if !isDeleted {
			data[k] = v
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repo: database error: %w", err)
	}

	return data, nil
}

// Close correctly closes connections from the pool.
func (r *PostgresStorage) Close() error {
	r.pool.Close()
	return nil
}

// Ping is used for detecting database availability.
func (r *PostgresStorage) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

// DeleteBatch provides delete operation for a batch of records from postgre storage.
func (r *PostgresStorage) DeleteBatch(ctx context.Context, batch map[string][]string) error {
	if len(batch) == 0 {
		return nil
	}

	var userIDs []string
	var aliases []string

	for userID, userAliases := range batch {
		for _, alias := range userAliases {
			userIDs = append(userIDs, userID)
			aliases = append(aliases, alias)
		}
	}

	query := `
		UPDATE urls AS u
		SET is_deleted = true
		FROM (
			SELECT unnest($1::uuid[]) AS user_id, unnest($2::varchar[]) AS alias
		) AS data
		WHERE u.user_id = data.user_id
		  AND u.alias = data.alias
		  AND u.is_deleted = false;
	`

	_, err := r.pool.Exec(ctx, query, userIDs, aliases)
	if err != nil {
		return fmt.Errorf("repository: failed to execute batch update: %w", err)
	}

	return nil
}
