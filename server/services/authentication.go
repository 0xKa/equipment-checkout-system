package services

import (
	"context"

	"github.com/0xKa/equipment-checkout-system/server/types"
)

var roleCapabilities = map[string]types.CapabilitySet{
	string(types.UserRoleEmployee): types.NewCapabilitySet(
		types.CapabilityInventoryRead,
		types.CapabilityCheckoutSelf,
	),
	string(types.UserRoleInventoryAdmin): types.NewCapabilitySet(
		types.CapabilityInventoryRead,
		types.CapabilityInventoryManage,
		types.CapabilityUsersManage,
		types.CapabilityCheckoutSelf,
		types.CapabilityCheckoutManage,
		types.CapabilityCheckoutHistoryReadAll,
	),
	string(types.UserRoleAuditor): types.NewCapabilitySet(
		types.CapabilityInventoryRead,
		types.CapabilityCheckoutHistoryReadAll,
	),
}

type TokenVerifier interface {
	// Verifies a signed access token and returns its trusted identity claims.
	Verify(ctx context.Context, rawToken string) (types.VerifiedIdentity, error)
}

type AuthenticationService interface {
	// Authenticates a token as one active, pre-linked local actor.
	Authenticate(ctx context.Context, rawToken string) (types.Actor, error)
}

type authenticationService struct {
	verifier TokenVerifier
	identity IdentityResolver
}

var _ AuthenticationService = (*authenticationService)(nil)

// Builds the application authentication boundary.
func NewAuthenticationService(
	verifier TokenVerifier,
	identity IdentityResolver,
) AuthenticationService {
	return &authenticationService{
		verifier: verifier,
		identity: identity,
	}
}

func (s *authenticationService) Authenticate(
	ctx context.Context,
	rawToken string,
) (types.Actor, error) {
	identity, err := s.verifier.Verify(ctx, rawToken)
	if err != nil {
		return types.Actor{}, err
	}

	capabilities, recognized := capabilitiesForRoles(identity.Roles)
	if !recognized {
		return types.Actor{}, types.ErrForbidden
	}

	actor, err := s.identity.Resolve(ctx, identity)
	if err != nil {
		return types.Actor{}, err
	}

	actor.Capabilities = capabilities
	return actor, nil
}

func capabilitiesForRoles(roles []string) (types.CapabilitySet, bool) {
	var recognized types.CapabilitySet
	recognizedCount := 0

	for _, role := range roles {
		roleSet, ok := roleCapabilities[role]
		if !ok {
			continue
		}
		recognized = roleSet
		recognizedCount++
	}

	return recognized, recognizedCount == 1
}
