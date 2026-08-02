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

	subject, err := s.identities.CreateIdentity(ctx, profile)
	if err != nil {
		return types.User{}, err
	}

	if err := s.identities.ReplaceRole(ctx, subject, input.Role); err != nil {
		s.deleteCreatedIdentity(subject)
		return types.User{}, err
	}

	row, err := s.queries.CreateUser(ctx, sqlcgen.CreateUserParams{
		Username:        profile.Username,
		Email:           profile.Email,
		DisplayName:     profile.DisplayName,
		Role:            string(input.Role),
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

	var previous sqlcgen.User
	var updated sqlcgen.User
	identityChanged := false
	runErr := s.transactions.Run(ctx, func(queries sqlcgen.Querier) error {
		var err error
		previous, err = lockedLinkedUser(ctx, queries, id, s.issuer)
		if err != nil {
			return err
		}
		if err := validateUniqueUserIdentity(ctx, queries, profile, id); err != nil {
			return err
		}

		if err := s.identities.UpdateProfile(ctx, *previous.ExternalSubject, profile); err != nil {
			return err
		}
		identityChanged = true

		updated, err = queries.UpdateUser(ctx, sqlcgen.UpdateUserParams{
			Username:    profile.Username,
			Email:       profile.Email,
			DisplayName: profile.DisplayName,
			ID:          id,
		})
		if err != nil {
			return mapUserWriteError(ctx, "update user profile", err)
		}
		return nil
	})
	if runErr != nil {
		if identityChanged {
			s.restoreProfile(previous)
		}
		return types.User{}, fmt.Errorf("update user profile: %w", runErr)
	}

	return userFromRow(updated), nil
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

	var previous sqlcgen.User
	var updated sqlcgen.User
	identityChanged := false
	runErr := s.transactions.Run(ctx, func(queries sqlcgen.Querier) error {
		var err error
		previous, err = lockedLinkedUser(ctx, queries, id, s.issuer)
		if err != nil {
			return err
		}

		if err := s.identities.ReplaceRole(ctx, *previous.ExternalSubject, role); err != nil {
			return err
		}
		identityChanged = true

		updated, err = queries.SetUserRole(ctx, sqlcgen.SetUserRoleParams{
			Role: string(role),
			ID:   id,
		})
		if err != nil {
			return mapUserWriteError(ctx, "set user role", err)
		}
		return nil
	})
	if runErr != nil {
		if identityChanged {
			s.restoreRole(previous)
		}
		return types.User{}, fmt.Errorf("set user role: %w", runErr)
	}

	return userFromRow(updated), nil
}

func (s *userService) SetActive(
	ctx context.Context,
	id int64,
	isActive bool,
) (types.User, error) {
	if err := validateUserID(id); err != nil {
		return types.User{}, err
	}

	var previous sqlcgen.User
	var updated sqlcgen.User
	identityChanged := false
	runErr := s.transactions.Run(ctx, func(queries sqlcgen.Querier) error {
		var err error
		previous, err = lockedLinkedUser(ctx, queries, id, s.issuer)
		if err != nil {
			return err
		}

		if err := s.identities.SetEnabled(ctx, *previous.ExternalSubject, isActive); err != nil {
			return err
		}
		identityChanged = true

		updated, err = queries.SetUserActive(ctx, sqlcgen.SetUserActiveParams{
			IsActive: isActive,
			ID:       id,
		})
		if err != nil {
			return utils.UnexpectedDatabaseError(ctx, "set user active status", err)
		}
		return nil
	})
	if runErr != nil {
		if identityChanged {
			s.restoreEnabled(previous)
		}
		return types.User{}, fmt.Errorf("set user active status: %w", runErr)
	}

	return userFromRow(updated), nil
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

func (s *userService) deleteCreatedIdentity(subject string) {
	if err := s.identities.DeleteIdentity(context.Background(), subject); err != nil {
		s.log.Warn("user creation compensation failed; run reconcile-users")
	}
}

func (s *userService) restoreProfile(previous sqlcgen.User) {
	err := s.identities.UpdateProfile(context.Background(), *previous.ExternalSubject, userProfile(previous))
	if err != nil {
		s.logCompensationFailure(previous.ID, "profile")
	}
}

func (s *userService) restoreRole(previous sqlcgen.User) {
	err := s.identities.ReplaceRole(
		context.Background(),
		*previous.ExternalSubject,
		types.UserRole(previous.Role),
	)
	if err != nil {
		s.logCompensationFailure(previous.ID, "role")
	}
}

func (s *userService) restoreEnabled(previous sqlcgen.User) {
	err := s.identities.SetEnabled(
		context.Background(),
		*previous.ExternalSubject,
		previous.IsActive,
	)
	if err != nil {
		s.logCompensationFailure(previous.ID, "activation")
	}
}

func (s *userService) logCompensationFailure(userID int64, field string) {
	s.log.Warn(
		"user synchronization compensation failed; run reconcile-users",
		zap.Int64("user_id", userID),
		zap.String("field", field),
	)
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
