package services

import (
	"context"

	"github.com/0xKa/equipment-checkout-system/server/types"
)

type ActorResolver interface {
	// Returns the active user represented by an actor ID.
	Resolve(ctx context.Context, userID int64) (types.Actor, error)
}

type actorResolver struct {
	users UserService
}

var _ ActorResolver = (*actorResolver)(nil)

// Builds an actor resolver backed by local users.
func NewActorResolver(users UserService) ActorResolver {
	return &actorResolver{users: users}
}

// Loads an active local user as an actor.
func (r *actorResolver) Resolve(ctx context.Context, userID int64) (types.Actor, error) {
	user, err := r.users.GetByID(ctx, userID)
	if err != nil {
		return types.Actor{}, err
	}
	if !user.IsActive {
		return types.Actor{}, types.ErrActorInactive
	}

	return types.Actor{User: user}, nil
}
