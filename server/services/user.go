package services

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"github.com/0xKa/equipment-checkout-system/server/db/sqlcgen"
	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/0xKa/equipment-checkout-system/server/utils"
	"github.com/jackc/pgx/v5"
)

const (
	constraintUsersUsername = "uq_users_username_normalized"
	constraintUsersEmail    = "uq_users_email_normalized"
)

type UserService interface {
	// Validates and persists a new active user.
	Create(ctx context.Context, input types.CreateUserRequest) (types.User, error)
	// Returns users matching an optional active-state filter.
	List(ctx context.Context, isActive *bool) ([]types.User, error)
	// Returns one user by ID.
	GetByID(ctx context.Context, id int64) (types.User, error)
	// Replaces editable profile fields while preserving status.
	Update(ctx context.Context, id int64, input types.UpdateUserRequest) (types.User, error)
	// Changes only the user's activation status.
	SetActive(ctx context.Context, id int64, isActive bool) (types.User, error)
}

type userService struct {
	queries sqlcgen.Querier
}

var _ UserService = (*userService)(nil)

// Builds a user service backed by database queries.
func NewUserService(queries sqlcgen.Querier) UserService {
	return &userService{queries: queries}
}

// Validates and persists a new user.
func (s *userService) Create(ctx context.Context, input types.CreateUserRequest) (types.User, error) {
	if err := ctx.Err(); err != nil {
		return types.User{}, err
	}

	input, err := normalizeUserInput(input)
	if err != nil {
		return types.User{}, err
	}
	if err := s.validateUniqueIdentity(ctx, input.Username, input.Email, 0); err != nil {
		return types.User{}, err
	}

	row, err := s.queries.CreateUser(ctx, sqlcgen.CreateUserParams{
		Username:    input.Username,
		Email:       input.Email,
		DisplayName: input.DisplayName,
	})
	if err != nil {
		return types.User{}, mapUserWriteError(ctx, "create user", err)
	}

	return userFromRow(row), nil
}

// Loads users with an optional active-state filter.
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

// Loads the user identified by a valid ID.
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

// Validates and replaces editable profile fields.
func (s *userService) Update(
	ctx context.Context,
	id int64,
	input types.UpdateUserRequest,
) (types.User, error) {
	if err := ctx.Err(); err != nil {
		return types.User{}, err
	}
	if err := validateUserID(id); err != nil {
		return types.User{}, err
	}

	input, err := normalizeUserInput(input)
	if err != nil {
		return types.User{}, err
	}
	if _, err := s.GetByID(ctx, id); err != nil {
		return types.User{}, err
	}
	if err := s.validateUniqueIdentity(ctx, input.Username, input.Email, id); err != nil {
		return types.User{}, err
	}

	row, err := s.queries.UpdateUser(ctx, sqlcgen.UpdateUserParams{
		Username:    input.Username,
		Email:       input.Email,
		DisplayName: input.DisplayName,
		ID:          id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return types.User{}, types.ErrUserNotFound
	}
	if err != nil {
		return types.User{}, mapUserWriteError(ctx, "update user", err)
	}

	return userFromRow(row), nil
}

// Changes status without modifying profile fields.
func (s *userService) SetActive(ctx context.Context, id int64, isActive bool) (types.User, error) {
	if err := ctx.Err(); err != nil {
		return types.User{}, err
	}
	if err := validateUserID(id); err != nil {
		return types.User{}, err
	}

	row, err := s.queries.SetUserActive(ctx, sqlcgen.SetUserActiveParams{
		IsActive: isActive,
		ID:       id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return types.User{}, types.ErrUserNotFound
	}
	if err != nil {
		return types.User{}, utils.UnexpectedDatabaseError(ctx, "set user active status", err)
	}

	return userFromRow(row), nil
}

// Checks normalized username and email availability.
func (s *userService) validateUniqueIdentity(
	ctx context.Context,
	username string,
	email *string,
	excludedID int64,
) error {
	exists, err := s.queries.UsernameExists(ctx, sqlcgen.UsernameExistsParams{
		Username:   username,
		ExcludedID: excludedID,
	})
	if err != nil {
		return utils.UnexpectedDatabaseError(ctx, "check username", err)
	}
	if exists {
		return types.ErrUsernameConflict
	}

	if email != nil {
		exists, err := s.queries.UserEmailExists(ctx, sqlcgen.UserEmailExistsParams{
			Email:      *email,
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

// Converts a database user into the public model.
func userFromRow(row sqlcgen.User) types.User {
	return types.User{
		ID:          row.ID,
		Username:    row.Username,
		Email:       row.Email,
		DisplayName: row.DisplayName,
		IsActive:    row.IsActive,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}

// Normalizes and validates editable profile fields.
func normalizeUserInput(input types.CreateUserRequest) (types.CreateUserRequest, error) {
	username := strings.TrimSpace(input.Username)
	if username == "" {
		return types.CreateUserRequest{}, types.ErrInvalidUserInput
	}

	email, err := normalizeOptionalUserValue(input.Email)
	if err != nil {
		return types.CreateUserRequest{}, types.ErrInvalidUserInput
	}
	if email != nil {
		normalizedEmail := strings.ToLower(*email)
		address, parseErr := mail.ParseAddress(normalizedEmail)
		if parseErr != nil || address.Address != normalizedEmail {
			return types.CreateUserRequest{}, types.ErrInvalidUserInput
		}
		email = &normalizedEmail
	}

	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		return types.CreateUserRequest{}, types.ErrInvalidUserInput
	}

	return types.CreateUserRequest{
		Username:    username,
		Email:       email,
		DisplayName: displayName,
	}, nil
}

// Preserves absence while rejecting a supplied blank value.
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

// Converts known database conflicts into domain errors.
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
	default:
		return utils.UnexpectedDatabaseError(ctx, operation, err)
	}
}

// Rejects nonpositive user IDs.
func validateUserID(id int64) error {
	if !utils.IsValidID(id) {
		return types.ErrInvalidUserID
	}

	return nil
}
