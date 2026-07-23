package utils

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	PostgresForeignKeyViolation = "23503"
	PostgresUniqueViolation     = "23505"
	PostgresCheckViolation      = "23514"
)

// PostgresError unwraps a PostgreSQL server error when one is present.
func PostgresError(err error) (*pgconn.PgError, bool) {
	var pgError *pgconn.PgError
	if !errors.As(err, &pgError) {
		return nil, false
	}

	return pgError, true
}

func PostgresErrorHasCode(err error, code string) bool {
	pgError, ok := PostgresError(err)
	return ok && pgError.Code == code
}

// UnexpectedDatabaseError preserves context cancellation and adds operation
// context to all other database errors.
func UnexpectedDatabaseError(ctx context.Context, operation string, err error) error {
	if contextErr := ctx.Err(); contextErr != nil {
		return contextErr
	}

	return fmt.Errorf("%s: %w", operation, err)
}
