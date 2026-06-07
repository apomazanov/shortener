package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/apomazanov/shortener/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type pgxPooler interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Ping(ctx context.Context) error
	Close()
}

type PostgresConfig interface {
	GetDatabaseDSN() string
	GetNoDBMigration() bool
}

type PostgresStorage struct {
	pool pgxPooler
}

/* -------------------------------------------------------------------------- */
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

/* -------------------------------------------------------------------------- */
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

/* -------------------------------------------------------------------------- */
func (r *PostgresStorage) Save(ctx context.Context, alias string, original string) (usedAlias string, err error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("repo: save aborted: %w", err)
	}

	queryCtx := context.WithoutCancel(ctx)
	queryCtx, cancel := context.WithTimeout(queryCtx, 3*time.Second)
	defer cancel()

	// check if original URL already saved

	var existingAlias string
	query := `SELECT alias FROM urls WHERE original = $1`
	err = r.pool.QueryRow(queryCtx, query, original).Scan(&existingAlias)
	if err == nil {
		return existingAlias, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("repo: database error: %w", err)
	}

	// creating new db record

	query = `
		INSERT INTO urls (alias, original)
		VALUES ($1, $2)
		ON CONFLICT (alias) DO NOTHING;
	`

	cmdTag, err := r.pool.Exec(queryCtx, query, alias, original)
	if err != nil {
		return "", fmt.Errorf("repo: database error: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return "", domain.ErrDuplicate
	}

	return alias, nil
}

/* -------------------------------------------------------------------------- */
func (r *PostgresStorage) Get(ctx context.Context, alias string) (original string, err error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("repo: get aborted: %w", err)
	}

	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `
		SELECT original 
		FROM urls 
		WHERE alias = $1;
	`

	err = r.pool.QueryRow(queryCtx, query, alias).Scan(&original)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", domain.ErrNotFound
		}
		return "", fmt.Errorf("repo: database error: %w", err)
	}

	return original, nil
}

/* -------------------------------------------------------------------------- */
func (r *PostgresStorage) Close() error {
	r.pool.Close()
	return nil
}

/* -------------------------------------------------------------------------- */
func (r *PostgresStorage) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}
