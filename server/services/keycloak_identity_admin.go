package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/Nerzal/gocloak/v14"
)

const (
	displayNameAttribute = "equipment_display_name"
	identityListPageSize = 100
)

type KeycloakIdentityAdminConfig struct {
	BaseURL             string
	Realm               string
	ServiceClientID     string
	ServiceClientSecret string
	Timeout             time.Duration
}

type keycloakIdentityAdmin struct {
	client gocloak.GoCloakIface
	cfg    KeycloakIdentityAdminConfig
}

var _ IdentityAdmin = (*keycloakIdentityAdmin)(nil)

func NewKeycloakIdentityAdmin(cfg KeycloakIdentityAdminConfig) IdentityAdmin {
	client := gocloak.NewClient(cfg.BaseURL)
	client.RestyClient().SetTimeout(cfg.Timeout)
	return &keycloakIdentityAdmin{client: client, cfg: cfg}
}

func (a *keycloakIdentityAdmin) CreateIdentity(
	ctx context.Context,
	profile types.IdentityProfile,
) (string, error) {
	var subject string
	err := a.withToken(ctx, func(callCtx context.Context, token string) error {
		created, err := a.client.CreateUser(callCtx, token, a.cfg.Realm, gocloak.User{
			Username:      stringPointer(profile.Username),
			Email:         profile.Email,
			Enabled:       boolPointer(true),
			EmailVerified: boolPointer(false),
			Attributes: map[string][]string{
				displayNameAttribute: {profile.DisplayName},
			},
		})
		if err != nil {
			return mapIdentityAdminError(err)
		}
		if strings.TrimSpace(created) == "" {
			return types.ErrIdentityAdminUnavailable
		}
		subject = created
		return nil
	})
	return subject, err
}

func (a *keycloakIdentityAdmin) UpdateProfile(
	ctx context.Context,
	subject string,
	profile types.IdentityProfile,
) error {
	return a.withToken(ctx, func(callCtx context.Context, token string) error {
		err := a.client.UpdateUser(callCtx, token, a.cfg.Realm, gocloak.User{
			ID:       stringPointer(subject),
			Username: stringPointer(profile.Username),
			Email:    updateEmailPointer(profile.Email),
			Attributes: map[string][]string{
				displayNameAttribute: {profile.DisplayName},
			},
		})
		return mapIdentityAdminError(err)
	})
}

func (a *keycloakIdentityAdmin) ReplaceRole(
	ctx context.Context,
	subject string,
	role types.UserRole,
) error {
	if !role.Valid() {
		return types.ErrInvalidUserRole
	}

	return a.withToken(ctx, func(callCtx context.Context, token string) error {
		current, err := a.client.GetRealmRolesByUserID(
			callCtx, token, a.cfg.Realm, subject,
		)
		if err != nil {
			return mapIdentityAdminError(err)
		}

		desired, err := a.client.GetRealmRole(
			callCtx, token, a.cfg.Realm, string(role),
		)
		if err != nil {
			return mapIdentityAdminError(err)
		}
		if desired == nil || desired.ID == nil || desired.Name == nil {
			return types.ErrIdentityAdminUnavailable
		}

		managed := concreteManagedRoles(current)
		if len(managed) == 1 && managed[0].Name != nil &&
			*managed[0].Name == string(role) {
			return nil
		}

		if len(managed) > 0 {
			if err := a.client.DeleteRealmRoleFromUser(
				callCtx, token, a.cfg.Realm, subject, managed,
			); err != nil {
				return mapIdentityAdminError(err)
			}
		}

		if err := a.client.AddRealmRoleToUser(
			callCtx,
			token,
			a.cfg.Realm,
			subject,
			[]gocloak.Role{*desired},
		); err != nil {
			if len(managed) > 0 {
				_ = a.client.AddRealmRoleToUser(
					callCtx, token, a.cfg.Realm, subject, managed,
				)
			}
			return mapIdentityAdminError(err)
		}
		return nil
	})
}

func (a *keycloakIdentityAdmin) SetEnabled(
	ctx context.Context,
	subject string,
	enabled bool,
) error {
	return a.withToken(ctx, func(callCtx context.Context, token string) error {
		err := a.client.UpdateUser(callCtx, token, a.cfg.Realm, gocloak.User{
			ID:      stringPointer(subject),
			Enabled: boolPointer(enabled),
		})
		return mapIdentityAdminError(err)
	})
}

func (a *keycloakIdentityAdmin) SetTemporaryPassword(
	ctx context.Context,
	subject string,
	password string,
) error {
	return a.withToken(ctx, func(callCtx context.Context, token string) error {
		err := a.client.SetPassword(
			callCtx, token, subject, a.cfg.Realm, password, true,
		)
		return mapIdentityAdminError(err)
	})
}

func (a *keycloakIdentityAdmin) DeleteIdentity(ctx context.Context, subject string) error {
	return a.withToken(ctx, func(callCtx context.Context, token string) error {
		return mapIdentityAdminError(
			a.client.DeleteUser(callCtx, token, a.cfg.Realm, subject),
		)
	})
}

func (a *keycloakIdentityAdmin) ListIdentities(
	ctx context.Context,
) ([]types.ManagedIdentity, error) {
	identities := make([]types.ManagedIdentity, 0)
	err := a.withToken(ctx, func(callCtx context.Context, token string) error {
		for first := 0; ; first += identityListPageSize {
			users, err := a.client.GetUsers(callCtx, token, a.cfg.Realm, gocloak.GetUsersParams{
				First: intPointer(first),
				Max:   intPointer(identityListPageSize),
			})
			if err != nil {
				return mapIdentityAdminError(err)
			}

			for _, user := range users {
				if user == nil || user.ID == nil || user.Username == nil ||
					user.ServiceAccountClientID != nil {
					continue
				}
				identities = append(identities, types.ManagedIdentity{
					Subject:  *user.ID,
					Username: *user.Username,
					Email:    user.Email,
				})
			}

			if len(users) < identityListPageSize {
				return nil
			}
		}
	})
	return identities, err
}

func (a *keycloakIdentityAdmin) withToken(
	ctx context.Context,
	operation func(context.Context, string) error,
) error {
	callCtx, cancel := context.WithTimeout(ctx, a.cfg.Timeout)
	defer cancel()

	token, err := a.client.LoginClient(
		callCtx,
		a.cfg.ServiceClientID,
		a.cfg.ServiceClientSecret,
		a.cfg.Realm,
	)
	if err != nil {
		return mapIdentityAdminError(err)
	}
	if token == nil || strings.TrimSpace(token.AccessToken) == "" {
		return types.ErrIdentityAdminUnavailable
	}
	return operation(callCtx, token.AccessToken)
}

func concreteManagedRoles(roles []*gocloak.Role) []gocloak.Role {
	result := make([]gocloak.Role, 0, len(roles))
	for _, role := range roles {
		if role != nil && role.Name != nil && types.UserRole(*role.Name).Valid() {
			result = append(result, *role)
		}
	}
	return result
}

func mapIdentityAdminError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	var apiError *gocloak.APIError
	if errors.As(err, &apiError) {
		switch apiError.Code {
		case http.StatusConflict:
			return fmt.Errorf("%w: %v", types.ErrIdentityAdminConflict, err)
		case http.StatusNotFound:
			return fmt.Errorf("%w: %v", types.ErrIdentityAdminNotFound, err)
		case http.StatusBadRequest:
			return fmt.Errorf("%w: %v", types.ErrIdentityAdminRejected, err)
		}
	}

	return fmt.Errorf("%w: %v", types.ErrIdentityAdminUnavailable, err)
}

func stringPointer(value string) *string { return &value }
func boolPointer(value bool) *bool       { return &value }
func intPointer(value int) *int          { return &value }

func updateEmailPointer(email *string) *string {
	if email != nil {
		return email
	}
	return stringPointer("")
}
