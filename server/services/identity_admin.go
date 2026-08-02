package services

import (
	"context"

	"github.com/0xKa/equipment-checkout-system/server/types"
)

// IdentityAdmin is the narrow boundary used to manage application identities.
// Implementations must not expose provider-specific types to callers.
type IdentityAdmin interface {
	Create(ctx context.Context, state types.IdentityState) (string, error)
	Replace(ctx context.Context, subject string, state types.IdentityState) error
	SetTemporaryPassword(ctx context.Context, subject, password string) error
	Delete(ctx context.Context, subject string) error
	List(ctx context.Context) ([]types.ManagedIdentity, error)
}
