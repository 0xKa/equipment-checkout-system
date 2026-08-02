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
	// Resolves only an exact pre-linked external identity as a local actor.
	Resolve(ctx context.Context, identity types.VerifiedIdentity) (types.Actor, error)
}

type identityResolver struct {
	queries sqlcgen.Querier
}

var _ IdentityResolver = (*identityResolver)(nil)

func NewIdentityResolver(queries sqlcgen.Querier) IdentityResolver {
	return &identityResolver{queries: queries}
}

func (r *identityResolver) Resolve(
	ctx context.Context,
	identity types.VerifiedIdentity,
) (types.Actor, error) {
	if err := ctx.Err(); err != nil {
		return types.Actor{}, err
	}

	issuer := identity.Issuer
	subject := identity.Subject
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
