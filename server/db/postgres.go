package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PoolOptions struct {
	MaxConnections        int32
	MinConnections        int32
	MaxConnectionLifetime time.Duration
}

func NewPool(
	ctx context.Context,
	databaseURL string,
	options PoolOptions,
) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, errors.New("DATABASE_URL is invalid")
	}

	poolConfig.MaxConns = options.MaxConnections
	poolConfig.MinConns = options.MinConnections
	poolConfig.MaxConnLifetime = options.MaxConnectionLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, errors.New("create database pool")
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, errors.New("database is unavailable")
	}

	return pool, nil
}
