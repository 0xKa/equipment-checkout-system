package types

import "context"

// Actor represents the active local user performing an attributed operation.
type Actor struct {
	User User
}

type actorContextKey struct{}

// Stores the resolved actor in a derived context.
func ContextWithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorContextKey{}, actor)
}

// Retrieves the resolved actor when present.
func ActorFromContext(ctx context.Context) (Actor, bool) {
	actor, ok := ctx.Value(actorContextKey{}).(Actor)
	return actor, ok
}
