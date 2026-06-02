package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxPooler interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Ping(ctx context.Context) error
	Close()
}

type PostgresConfig interface {
	GetDatabaseDSN() string
}

type PostgresStorage struct {
	pool pgxPooler
}

/* -------------------------------------------------------------------------- */
func NewPostgresStorage(cfg PostgresConfig) (*PostgresStorage, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(cfg.GetDatabaseDSN())
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
func (r *PostgresStorage) Save(ctx context.Context, alias string, original string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("repo: save aborted: %w", err)
	}

	queryCtx := context.WithoutCancel(ctx)
	queryCtx, cancel := context.WithTimeout(queryCtx, 3*time.Second)
	defer cancel()

	query := `
		INSERT INTO urls (alias, original)
		VALUES ($1, $2)
		ON CONFLICT (alias) DO NOTHING;
	`

	cmdTag, err := r.pool.Exec(queryCtx, query, alias, original)
	if err != nil {
		return fmt.Errorf("repo: database error: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return domain.ErrDuplicate
	}

	return nil
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
