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

func (a *keycloakIdentityAdmin) Create(
	ctx context.Context,
	state types.IdentityState,
) (string, error) {
	if !state.Role.Valid() {
		return "", types.ErrInvalidUserRole
	}

	var subject string
	err := a.withToken(ctx, func(callCtx context.Context, token string) error {
		created, err := a.client.CreateUser(callCtx, token, a.cfg.Realm, gocloak.User{
			Username:      new(state.Profile.Username),
			Email:         state.Profile.Email,
			Enabled:       new(state.IsActive),
			EmailVerified: new(false),
			Attributes: map[string][]string{
				displayNameAttribute: {state.Profile.DisplayName},
			},
		})
		if err != nil {
			return mapIdentityAdminError(err)
		}
		if strings.TrimSpace(created) == "" {
			return types.ErrIdentityAdminUnavailable
		}

		subject = created
		return a.replaceRole(callCtx, token, subject, state.Role)
	})
	return subject, err
}

func (a *keycloakIdentityAdmin) Replace(
	ctx context.Context,
	subject string,
	state types.IdentityState,
) error {
	if !state.Role.Valid() {
		return types.ErrInvalidUserRole
	}

	return a.withToken(ctx, func(callCtx context.Context, token string) error {
		err := a.client.UpdateUser(callCtx, token, a.cfg.Realm, gocloak.User{
			ID:       new(subject),
			Username: new(state.Profile.Username),
			Email:    updateEmailPointer(state.Profile.Email),
			Enabled:  new(state.IsActive),
			Attributes: map[string][]string{
				displayNameAttribute: {state.Profile.DisplayName},
			},
		})
		if err != nil {
			return mapIdentityAdminError(err)
		}
		return a.replaceRole(callCtx, token, subject, state.Role)
	})
}

func (a *keycloakIdentityAdmin) replaceRole(
	ctx context.Context,
	token string,
	subject string,
	role types.UserRole,
) error {
	current, err := a.client.GetRealmRolesByUserID(
		ctx, token, a.cfg.Realm, subject,
	)
	if err != nil {
		return mapIdentityAdminError(err)
	}

	desired, err := a.client.GetRealmRole(
		ctx, token, a.cfg.Realm, string(role),
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
			ctx, token, a.cfg.Realm, subject, managed,
		); err != nil {
			return mapIdentityAdminError(err)
		}
	}

	return mapIdentityAdminError(a.client.AddRealmRoleToUser(
		ctx,
		token,
		a.cfg.Realm,
		subject,
		[]gocloak.Role{*desired},
	))
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

func (a *keycloakIdentityAdmin) Delete(ctx context.Context, subject string) error {
	return a.withToken(ctx, func(callCtx context.Context, token string) error {
		return mapIdentityAdminError(
			a.client.DeleteUser(callCtx, token, a.cfg.Realm, subject),
		)
	})
}

func (a *keycloakIdentityAdmin) List(
	ctx context.Context,
) ([]types.ManagedIdentity, error) {
	identities := make([]types.ManagedIdentity, 0)
	err := a.withToken(ctx, func(callCtx context.Context, token string) error {
		for first := 0; ; first += identityListPageSize {
			users, err := a.client.GetUsers(callCtx, token, a.cfg.Realm, gocloak.GetUsersParams{
				First: new(first),
				Max:   new(identityListPageSize),
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

func updateEmailPointer(email *string) *string {
	if email != nil {
		return email
	}
	return new("")
}
