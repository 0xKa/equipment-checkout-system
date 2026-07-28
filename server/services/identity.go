package services

import (
	"context"
	"errors"
	"strings"

	"github.com/0xKa/equipment-checkout-system/server/db/sqlcgen"
	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/0xKa/equipment-checkout-system/server/utils"
	"github.com/jackc/pgx/v5"
)

type IdentityResolver interface {
	// Resolves or safely provisions an exact external identity as a local actor.
	Resolve(ctx context.Context, identity types.VerifiedIdentity) (types.Actor, error)
}

type identityResolver struct {
	queries sqlcgen.Querier
}

var _ IdentityResolver = (*identityResolver)(nil)

// Builds an external identity resolver backed by local users.
func NewIdentityResolver(queries sqlcgen.Querier) IdentityResolver {
	return &identityResolver{queries: queries}
}

// Loads an exact linked actor or provisions a new local profile from trusted
// token claims without linking by username or email.
func (r *identityResolver) Resolve(
	ctx context.Context,
	identity types.VerifiedIdentity,
) (types.Actor, error) {
	if err := ctx.Err(); err != nil {
		return types.Actor{}, err
	}

	row, err := r.getByExternalIdentity(ctx, identity.Issuer, identity.Subject)
	if err == nil {
		return actorFromIdentityRow(row)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return types.Actor{}, utils.UnexpectedDatabaseError(ctx, "resolve external identity", err)
	}

	input, err := jitUserInput(identity)
	if err != nil {
		return types.Actor{}, err
	}

	row, err = r.queries.CreateUserWithExternalIdentity(
		ctx,
		sqlcgen.CreateUserWithExternalIdentityParams{
			Username:        input.Username,
			Email:           input.Email,
			DisplayName:     input.DisplayName,
			IdentityIssuer:  &identity.Issuer,
			ExternalSubject: &identity.Subject,
		},
	)
	if err == nil {
		return actorFromIdentityRow(row)
	}

	pgError, ok := utils.PostgresError(err)
	if !ok || pgError.Code != utils.PostgresUniqueViolation {
		return types.Actor{}, utils.UnexpectedDatabaseError(ctx, "provision external identity", err)
	}

	// A concurrent request may win any of the overlapping unique constraints.
	// Exact identity always takes precedence over the reported constraint.
	row, lookupErr := r.getByExternalIdentity(ctx, identity.Issuer, identity.Subject)
	if lookupErr == nil {
		return actorFromIdentityRow(row)
	}
	if !errors.Is(lookupErr, pgx.ErrNoRows) {
		return types.Actor{}, utils.UnexpectedDatabaseError(
			ctx,
			"resolve concurrently provisioned identity",
			lookupErr,
		)
	}

	switch pgError.ConstraintName {
	case constraintUsersUsername, constraintUsersEmail:
		return types.Actor{}, types.ErrIdentityConflict
	default:
		return types.Actor{}, utils.UnexpectedDatabaseError(ctx, "provision external identity", err)
	}
}

func (r *identityResolver) getByExternalIdentity(
	ctx context.Context,
	issuer string,
	subject string,
) (sqlcgen.User, error) {
	return r.queries.GetUserByExternalIdentity(
		ctx,
		sqlcgen.GetUserByExternalIdentityParams{
			IdentityIssuer:  &issuer,
			ExternalSubject: &subject,
		},
	)
}

func actorFromIdentityRow(row sqlcgen.User) (types.Actor, error) {
	user := userFromRow(row)
	if !user.IsActive {
		return types.Actor{}, types.ErrAccountInactive
	}
	return types.Actor{User: user}, nil
}

func jitUserInput(identity types.VerifiedIdentity) (types.CreateUserRequest, error) {
	displayName := identity.Name
	if strings.TrimSpace(displayName) == "" {
		displayName = identity.PreferredUsername
	}

	var email *string
	if identity.EmailVerified && strings.TrimSpace(identity.Email) != "" {
		email = &identity.Email
	}

	input, err := normalizeUserInput(types.CreateUserRequest{
		Username:    identity.PreferredUsername,
		Email:       email,
		DisplayName: displayName,
	})
	if err != nil {
		return types.CreateUserRequest{}, types.ErrIdentityProfileInvalid
	}
	return input, nil
}
