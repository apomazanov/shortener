package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresConfig interface {
	GetDatabaseDSN() string
}

type PostgresStorage struct {
	pool *pgxpool.Pool
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

	return nil
}

/* -------------------------------------------------------------------------- */
func (r *PostgresStorage) Get(ctx context.Context, alias string) (original string, err error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("repo: get aborted: %w", err)
	}

	return "", domain.ErrNotFound
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
