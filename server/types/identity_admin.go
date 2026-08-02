package types

// IdentityProfile is the application-owned profile mirrored to Keycloak.
type IdentityProfile struct {
	Username    string
	Email       *string
	DisplayName string
}

// IdentityState is the complete application-owned user state mirrored to the
// identity provider.
type IdentityState struct {
	Profile  IdentityProfile
	Role     UserRole
	IsActive bool
}

// ManagedIdentity is the safe subset needed for reconciliation reporting.
type ManagedIdentity struct {
	Subject  string
	Username string
	Email    *string
}
