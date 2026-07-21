package services

import (
	"context"
	"errors"
	"strings"

	"github.com/0xKa/equipment-checkout-system/server/db/sqlcgen"
	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/0xKa/equipment-checkout-system/server/utils"
	"github.com/jackc/pgx/v5"
)

const constraintCategoriesName = "uq_categories_name"

type CategoryService interface {
	Create(ctx context.Context, input types.CreateCategoryRequest) (types.Category, error)
	List(ctx context.Context) ([]types.Category, error)
	GetByID(ctx context.Context, id int64) (types.Category, error)
	Update(ctx context.Context, id int64, input types.UpdateCategoryRequest) (types.Category, error)
	Delete(ctx context.Context, id int64) error
}

type categoryService struct {
	queries sqlcgen.Querier
}

var _ CategoryService = (*categoryService)(nil)

func NewCategoryService(queries sqlcgen.Querier) CategoryService {
	return &categoryService{queries: queries}
}

func (s *categoryService) Create(ctx context.Context, input types.CreateCategoryRequest) (types.Category, error) {
	if err := ctx.Err(); err != nil {
		return types.Category{}, err
	}

	input, err := normalizeCategoryInput(input)
	if err != nil {
		return types.Category{}, err
	}

	exists, err := s.queries.CategoryNameExists(ctx, sqlcgen.CategoryNameExistsParams{
		Name:       input.Name,
		ExcludedID: 0,
	})
	if err != nil {
		return types.Category{}, utils.UnexpectedDatabaseError(ctx, "check category name", err)
	}
	if exists {
		return types.Category{}, types.ErrCategoryNameConflict
	}

	row, err := s.queries.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		Name:        input.Name,
		Description: input.Description,
	})
	if err != nil {
		return types.Category{}, mapCategoryWriteError(ctx, "create category", err)
	}

	return categoryFromRow(row), nil
}

func (s *categoryService) List(ctx context.Context) ([]types.Category, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rows, err := s.queries.ListCategories(ctx)
	if err != nil {
		return nil, utils.UnexpectedDatabaseError(ctx, "list categories", err)
	}

	categories := make([]types.Category, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, categoryFromRow(row))
	}

	return categories, nil
}

func (s *categoryService) GetByID(ctx context.Context, id int64) (types.Category, error) {
	if err := ctx.Err(); err != nil {
		return types.Category{}, err
	}
	if err := validateCategoryID(id); err != nil {
		return types.Category{}, err
	}

	return s.getByID(ctx, id)
}

func (s *categoryService) Update(ctx context.Context, id int64, input types.UpdateCategoryRequest) (types.Category, error) {
	if err := ctx.Err(); err != nil {
		return types.Category{}, err
	}
	if err := validateCategoryID(id); err != nil {
		return types.Category{}, err
	}

	input, err := normalizeCategoryInput(input)
	if err != nil {
		return types.Category{}, err
	}
	if _, err := s.getByID(ctx, id); err != nil {
		return types.Category{}, err
	}

	exists, err := s.queries.CategoryNameExists(ctx, sqlcgen.CategoryNameExistsParams{
		Name:       input.Name,
		ExcludedID: id,
	})
	if err != nil {
		return types.Category{}, utils.UnexpectedDatabaseError(ctx, "check category name", err)
	}
	if exists {
		return types.Category{}, types.ErrCategoryNameConflict
	}

	row, err := s.queries.UpdateCategory(ctx, sqlcgen.UpdateCategoryParams{
		Name:        input.Name,
		Description: input.Description,
		ID:          id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return types.Category{}, types.ErrCategoryNotFound
	}
	if err != nil {
		return types.Category{}, mapCategoryWriteError(ctx, "update category", err)
	}

	return categoryFromRow(row), nil
}

func (s *categoryService) Delete(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateCategoryID(id); err != nil {
		return err
	}

	_, err := s.queries.DeleteCategory(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return types.ErrCategoryNotFound
	}
	if utils.PostgresErrorHasCode(err, utils.PostgresForeignKeyViolation) {
		return types.ErrCategoryInUse
	}
	if err != nil {
		return utils.UnexpectedDatabaseError(ctx, "delete category", err)
	}

	return nil
}

func (s *categoryService) getByID(ctx context.Context, id int64) (types.Category, error) {
	row, err := s.queries.GetCategory(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return types.Category{}, types.ErrCategoryNotFound
	}
	if err != nil {
		return types.Category{}, utils.UnexpectedDatabaseError(ctx, "get category", err)
	}

	return categoryFromRow(row), nil
}

func categoryFromRow(row sqlcgen.Category) types.Category {
	return types.Category{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}

func mapCategoryWriteError(ctx context.Context, operation string, err error) error {
	if pgError, ok := utils.PostgresError(err); ok &&
		pgError.Code == utils.PostgresUniqueViolation &&
		pgError.ConstraintName == constraintCategoriesName {
		return types.ErrCategoryNameConflict
	}

	return utils.UnexpectedDatabaseError(ctx, operation, err)
}

func normalizeCategoryInput(input types.CreateCategoryRequest) (types.CreateCategoryRequest, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	if input.Name == "" {
		return types.CreateCategoryRequest{}, types.ErrInvalidCategoryInput
	}

	return input, nil
}

func validateCategoryID(id int64) error {
	if !utils.IsValidID(id) {
		return types.ErrInvalidCategoryID
	}

	return nil
}
