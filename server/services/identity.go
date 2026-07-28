package services

import (
	"context"
	"errors"

	"github.com/0xKa/equipment-checkout-system/server/db/sqlcgen"
	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/0xKa/equipment-checkout-system/server/utils"
	"github.com/jackc/pgx/v5"
)

type IdentityResolver interface {
	// Resolves an exact external identity to an active local actor.
	Resolve(ctx context.Context, issuer string, subject string) (types.Actor, error)
}

type identityResolver struct {
	queries sqlcgen.Querier
}

var _ IdentityResolver = (*identityResolver)(nil)

// Builds an external identity resolver backed by local users.
func NewIdentityResolver(queries sqlcgen.Querier) IdentityResolver {
	return &identityResolver{queries: queries}
}

// Loads the active local actor linked to an exact issuer and subject.
func (r *identityResolver) Resolve(
	ctx context.Context,
	issuer string,
	subject string,
) (types.Actor, error) {
	if err := ctx.Err(); err != nil {
		return types.Actor{}, err
	}

	row, err := r.queries.GetUserByExternalIdentity(
		ctx,
		sqlcgen.GetUserByExternalIdentityParams{
			IdentityIssuer:  &issuer,
			ExternalSubject: &subject,
		},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return types.Actor{}, types.ErrIdentityNotLinked
	}
	if err != nil {
		return types.Actor{}, utils.UnexpectedDatabaseError(ctx, "resolve external identity", err)
	}

	user := userFromRow(row)
	if !user.IsActive {
		return types.Actor{}, types.ErrAccountInactive
	}

	return types.Actor{User: user}, nil
}
