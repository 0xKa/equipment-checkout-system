package services

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/0xKa/equipment-checkout-system/server/db/sqlcgen"
	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/0xKa/equipment-checkout-system/server/utils"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

const (
	constraintUsersUsername = "uq_users_username_normalized"
	constraintUsersEmail    = "uq_users_email_normalized"
	constraintUsersRole     = "ck_users_role"
	maxIdentityFieldLength  = 255
)

type UserTransactionRunner interface {
	Run(ctx context.Context, fn func(sqlcgen.Querier) error) error
}

type UserService interface {
	Create(ctx context.Context, input types.CreateUserRequest) (types.User, error)
	List(ctx context.Context, isActive *bool) ([]types.User, error)
	GetByID(ctx context.Context, id int64) (types.User, error)
	UpdateProfile(
		ctx context.Context,
		id int64,
		input types.UpdateUserProfileRequest,
	) (types.User, error)
	SetRole(ctx context.Context, id int64, role types.UserRole) (types.User, error)
	SetActive(ctx context.Context, id int64, isActive bool) (types.User, error)
	Deprovision(ctx context.Context, id int64) error
	SetTemporaryPassword(ctx context.Context, id int64, password string) error
}

type userService struct {
	queries      sqlcgen.Querier
	transactions UserTransactionRunner
	identities   IdentityAdmin
	issuer       string
	log          *zap.Logger
}

var _ UserService = (*userService)(nil)

func NewUserService(
	queries sqlcgen.Querier,
	transactions UserTransactionRunner,
	identities IdentityAdmin,
	issuer string,
	log *zap.Logger,
) UserService {
	return &userService{
		queries:      queries,
		transactions: transactions,
		identities:   identities,
		issuer:       issuer,
		log:          log,
	}
}

func (s *userService) Create(
	ctx context.Context,
	input types.CreateUserRequest,
) (types.User, error) {
	if err := ctx.Err(); err != nil {
		return types.User{}, err
	}

	profile, err := normalizeUserProfile(input.Username, input.Email, input.DisplayName)
	if err != nil {
		return types.User{}, err
	}
	if !input.Role.Valid() {
		return types.User{}, types.ErrInvalidUserRole
	}
	if err := validateUniqueUserIdentity(ctx, s.queries, profile, 0); err != nil {
		return types.User{}, err
	}

	state := types.IdentityState{
		Profile:  profile,
		Role:     input.Role,
		IsActive: true,
	}
	// Creation writes the complete Keycloak identity first, then inserts the
	// linked local row. The local insert is one atomic statement, so it does not
	// need an explicit PostgreSQL transaction. If it fails, the service deletes
	// the newly created Keycloak identity before returning failure.
	subject, err := s.identities.Create(ctx, state)
	if err != nil {
		if subject != "" {
			s.deleteCreatedIdentity(subject)
		}
		return types.User{}, err
	}

	row, err := s.queries.CreateUser(ctx, sqlcgen.CreateUserParams{
		Username:        state.Profile.Username,
		Email:           state.Profile.Email,
		DisplayName:     state.Profile.DisplayName,
		Role:            string(state.Role),
		IdentityIssuer:  &s.issuer,
		ExternalSubject: &subject,
	})
	if err != nil {
		s.deleteCreatedIdentity(subject)
		return types.User{}, mapUserWriteError(ctx, "create user", err)
	}

	return userFromRow(row), nil
}

func (s *userService) List(ctx context.Context, isActive *bool) ([]types.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rows, err := s.queries.ListUsers(ctx, isActive)
	if err != nil {
		return nil, utils.UnexpectedDatabaseError(ctx, "list users", err)
	}

	users := make([]types.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, userFromRow(row))
	}
	return users, nil
}

func (s *userService) GetByID(ctx context.Context, id int64) (types.User, error) {
	if err := ctx.Err(); err != nil {
		return types.User{}, err
	}
	if err := validateUserID(id); err != nil {
		return types.User{}, err
	}

	row, err := s.queries.GetUser(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return types.User{}, types.ErrUserNotFound
	}
	if err != nil {
		return types.User{}, utils.UnexpectedDatabaseError(ctx, "get user", err)
	}
	return userFromRow(row), nil
}

func (s *userService) UpdateProfile(
	ctx context.Context,
	id int64,
	input types.UpdateUserProfileRequest,
) (types.User, error) {
	if err := validateUserID(id); err != nil {
		return types.User{}, err
	}
	profile, err := normalizeUserProfile(input.Username, input.Email, input.DisplayName)
	if err != nil {
		return types.User{}, err
	}

	return s.updateManagedUser(ctx, id, "update user profile", func(state *types.IdentityState) {
		state.Profile = profile
	})
}

func (s *userService) SetRole(
	ctx context.Context,
	id int64,
	role types.UserRole,
) (types.User, error) {
	if err := validateUserID(id); err != nil {
		return types.User{}, err
	}
	if !role.Valid() {
		return types.User{}, types.ErrInvalidUserRole
	}

	return s.updateManagedUser(ctx, id, "set user role", func(state *types.IdentityState) {
		state.Role = role
	})
}

func (s *userService) SetActive(
	ctx context.Context,
	id int64,
	isActive bool,
) (types.User, error) {
	if err := validateUserID(id); err != nil {
		return types.User{}, err
	}

	return s.updateManagedUser(ctx, id, "set user active status", func(state *types.IdentityState) {
		state.IsActive = isActive
	})
}

func (s *userService) Deprovision(ctx context.Context, id int64) error {
	_, err := s.SetActive(ctx, id, false)
	return err
}

func (s *userService) SetTemporaryPassword(
	ctx context.Context,
	id int64,
	password string,
) error {
	if err := validateUserID(id); err != nil {
		return err
	}
	if password == "" {
		return types.ErrInvalidPassword
	}

	row, err := s.queries.GetUser(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return types.ErrUserNotFound
	}
	if err != nil {
		return utils.UnexpectedDatabaseError(ctx, "get user for temporary password", err)
	}
	if err := requireManagedIdentity(row, s.issuer); err != nil {
		return err
	}

	return s.identities.SetTemporaryPassword(ctx, *row.ExternalSubject, password)
}

// updateManagedUser owns one logical user transaction across PostgreSQL and
// Keycloak. Keycloak cannot join the PostgreSQL transaction, so the service:
//
//  1. locks and snapshots the local row;
//  2. derives and validates the complete desired state;
//  3. replaces the Keycloak state;
//  4. writes the same state locally and commits PostgreSQL.
//
// TransactionManager.Run returns success only after the database commit. If
// Keycloak may have changed but the replacement, local write, or commit fails,
// the service makes one bounded attempt to restore the snapshot. A failed
// restoration is logged for the explicit reconciliation workflow.
func (s *userService) updateManagedUser(
	ctx context.Context,
	id int64,
	operation string,
	mutate func(*types.IdentityState),
) (types.User, error) {
	var previous sqlcgen.User
	var updated sqlcgen.User
	identityMayHaveChanged := false

	runErr := s.transactions.Run(ctx, func(queries sqlcgen.Querier) error {
		var err error
		previous, err = lockedLinkedUser(ctx, queries, id, s.issuer)
		if err != nil {
			return err
		}

		desired := identityState(previous)
		mutate(&desired)
		if err := validateUniqueUserIdentity(ctx, queries, desired.Profile, id); err != nil {
			return err
		}

		// Replace can make more than one Keycloak Admin API call. Mark the
		// identity before calling it because an error can follow a partial
		// provider update.
		identityMayHaveChanged = true
		if err := s.identities.Replace(ctx, *previous.ExternalSubject, desired); err != nil {
			return err
		}

		updated, err = queries.UpdateManagedUser(ctx, sqlcgen.UpdateManagedUserParams{
			Username:    desired.Profile.Username,
			Email:       desired.Profile.Email,
			DisplayName: desired.Profile.DisplayName,
			Role:        string(desired.Role),
			IsActive:    desired.IsActive,
			ID:          id,
		})
		if err != nil {
			return mapUserWriteError(ctx, operation, err)
		}
		return nil
	})
	if runErr == nil {
		return userFromRow(updated), nil
	}

	if identityMayHaveChanged {
		s.restoreIdentity(previous, operation)
	}
	return types.User{}, fmt.Errorf("%s: %w", operation, runErr)
}

func (s *userService) deleteCreatedIdentity(subject string) {
	if err := s.identities.Delete(context.Background(), subject); err != nil {
		s.log.Warn("user creation compensation failed; run reconcile-users")
	}
}

func (s *userService) restoreIdentity(previous sqlcgen.User, operation string) {
	err := s.identities.Replace(
		context.Background(),
		*previous.ExternalSubject,
		identityState(previous),
	)
	if err != nil {
		s.log.Warn(
			"user synchronization compensation failed; run reconcile-users",
			zap.Int64("user_id", previous.ID),
			zap.String("operation", operation),
		)
	}
}

func lockedLinkedUser(
	ctx context.Context,
	queries sqlcgen.Querier,
	id int64,
	issuer string,
) (sqlcgen.User, error) {
	row, err := queries.GetUserForUpdate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlcgen.User{}, types.ErrUserNotFound
	}
	if err != nil {
		return sqlcgen.User{}, utils.UnexpectedDatabaseError(ctx, "lock user", err)
	}
	if err := requireManagedIdentity(row, issuer); err != nil {
		return sqlcgen.User{}, err
	}
	return row, nil
}

func requireManagedIdentity(row sqlcgen.User, issuer string) error {
	if row.IdentityIssuer == nil || row.ExternalSubject == nil ||
		*row.IdentityIssuer != issuer || strings.TrimSpace(*row.ExternalSubject) == "" {
		return types.ErrUserIdentityUnlinked
	}
	return nil
}

func validateUniqueUserIdentity(
	ctx context.Context,
	queries sqlcgen.Querier,
	profile types.IdentityProfile,
	excludedID int64,
) error {
	exists, err := queries.UsernameExists(ctx, sqlcgen.UsernameExistsParams{
		Username:   profile.Username,
		ExcludedID: excludedID,
	})
	if err != nil {
		return utils.UnexpectedDatabaseError(ctx, "check username", err)
	}
	if exists {
		return types.ErrUsernameConflict
	}

	if profile.Email != nil {
		exists, err := queries.UserEmailExists(ctx, sqlcgen.UserEmailExistsParams{
			Email:      *profile.Email,
			ExcludedID: excludedID,
		})
		if err != nil {
			return utils.UnexpectedDatabaseError(ctx, "check user email", err)
		}
		if exists {
			return types.ErrUserEmailConflict
		}
	}
	return nil
}

func userFromRow(row sqlcgen.User) types.User {
	return types.User{
		ID:          row.ID,
		Username:    row.Username,
		Email:       row.Email,
		DisplayName: row.DisplayName,
		Role:        types.UserRole(row.Role),
		IsActive:    row.IsActive,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}

func userProfile(row sqlcgen.User) types.IdentityProfile {
	return types.IdentityProfile{
		Username:    row.Username,
		Email:       row.Email,
		DisplayName: row.DisplayName,
	}
}

func identityState(row sqlcgen.User) types.IdentityState {
	return types.IdentityState{
		Profile:  userProfile(row),
		Role:     types.UserRole(row.Role),
		IsActive: row.IsActive,
	}
}

func normalizeUserProfile(
	username string,
	emailValue *string,
	displayNameValue string,
) (types.IdentityProfile, error) {
	username = strings.TrimSpace(username)
	usernameLength := utf8.RuneCountInString(username)
	if usernameLength < 3 || usernameLength > maxIdentityFieldLength {
		return types.IdentityProfile{}, types.ErrInvalidUserInput
	}

	email, err := normalizeOptionalUserValue(emailValue)
	if err != nil {
		return types.IdentityProfile{}, types.ErrInvalidUserInput
	}
	if email != nil {
		normalizedEmail := strings.ToLower(*email)
		if utf8.RuneCountInString(normalizedEmail) > maxIdentityFieldLength {
			return types.IdentityProfile{}, types.ErrInvalidUserInput
		}
		address, parseErr := mail.ParseAddress(normalizedEmail)
		if parseErr != nil || address.Address != normalizedEmail {
			return types.IdentityProfile{}, types.ErrInvalidUserInput
		}
		email = &normalizedEmail
	}

	displayName := strings.TrimSpace(displayNameValue)
	if displayName == "" || utf8.RuneCountInString(displayName) > maxIdentityFieldLength {
		return types.IdentityProfile{}, types.ErrInvalidUserInput
	}

	return types.IdentityProfile{
		Username:    username,
		Email:       email,
		DisplayName: displayName,
	}, nil
}

func normalizeOptionalUserValue(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil, types.ErrInvalidUserInput
	}
	return &normalized, nil
}

func mapUserWriteError(ctx context.Context, operation string, err error) error {
	pgError, ok := utils.PostgresError(err)
	if !ok {
		return utils.UnexpectedDatabaseError(ctx, operation, err)
	}

	switch {
	case pgError.Code == utils.PostgresUniqueViolation && pgError.ConstraintName == constraintUsersUsername:
		return types.ErrUsernameConflict
	case pgError.Code == utils.PostgresUniqueViolation && pgError.ConstraintName == constraintUsersEmail:
		return types.ErrUserEmailConflict
	case pgError.Code == utils.PostgresCheckViolation && pgError.ConstraintName == constraintUsersRole:
		return types.ErrInvalidUserRole
	default:
		return utils.UnexpectedDatabaseError(ctx, operation, err)
	}
}

func validateUserID(id int64) error {
	if !utils.IsValidID(id) {
		return types.ErrInvalidUserID
	}
	return nil
}
