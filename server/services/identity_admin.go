package services

import (
	"context"

	"github.com/0xKa/equipment-checkout-system/server/types"
)

// IdentityAdmin is the narrow boundary used to manage application identities.
// Implementations must not expose provider-specific types to callers.
type IdentityAdmin interface {
	CreateIdentity(ctx context.Context, profile types.IdentityProfile) (string, error)
	UpdateProfile(ctx context.Context, subject string, profile types.IdentityProfile) error
	ReplaceRole(ctx context.Context, subject string, role types.UserRole) error
	SetEnabled(ctx context.Context, subject string, enabled bool) error
	SetTemporaryPassword(ctx context.Context, subject, password string) error
	DeleteIdentity(ctx context.Context, subject string) error
	ListIdentities(ctx context.Context) ([]types.ManagedIdentity, error)
}
