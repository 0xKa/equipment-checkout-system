package types

// IdentityProfile is the application-owned profile mirrored to Keycloak.
type IdentityProfile struct {
	Username    string
	Email       *string
	DisplayName string
}

// ManagedIdentity is the safe subset needed for reconciliation reporting.
type ManagedIdentity struct {
	Subject  string
	Username string
	Email    *string
}
