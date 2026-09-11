package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/THEcanon001/turnero/internal/platform/config"
)

// New creates a new pgxpool connection pool from config and verifies connectivity.
func New(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	return NewFromDSN(ctx, cfg.DSN(), cfg.MaxOpenConns, cfg.MaxIdleConns)
}

// NewFromDSN creates a new pgxpool connection pool from a raw DSN string and verifies connectivity.
func NewFromDSN(ctx context.Context, dsn string, maxConns, minConns int) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("database: parse config: %w", err)
	}

	poolCfg.MaxConns = int32(maxConns)
	poolCfg.MinConns = int32(minConns)

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("database: connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database: ping: %w", err)
	}

	return pool, nil
}
