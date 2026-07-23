package db

import (
	"context"
	"errors"
	"time"

	"github.com/0xKa/equipment-checkout-system/server/db/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TransactionManager runs a callback against transaction-bound sqlc queries.
type TransactionManager struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
}

func NewTransactionManager(pool *pgxpool.Pool, queries *sqlcgen.Queries) *TransactionManager {
	return &TransactionManager{
		pool:    pool,
		queries: queries,
	}
}

func (m *TransactionManager) Run(
	ctx context.Context,
	fn func(sqlcgen.Querier) error,
) (runErr error) {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if runErr == nil {
			return
		}
		rollbackContext, cancelRollback := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelRollback()
		if rollbackErr := tx.Rollback(rollbackContext); rollbackErr != nil &&
			!errors.Is(rollbackErr, pgx.ErrTxClosed) {
			runErr = errors.Join(runErr, rollbackErr)
		}
	}()

	if err := fn(m.queries.WithTx(tx)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
