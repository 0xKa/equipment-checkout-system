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

const (
	constraintItemsAssetTag = "uq_items_asset_tag"
	constraintItemsSerial   = "uq_items_serial_number"
)

type ItemService interface {
	Create(ctx context.Context, input types.CreateItemRequest) (types.Item, error)
	List(ctx context.Context) ([]types.Item, error)
	GetByID(ctx context.Context, id int64) (types.Item, error)
	Update(ctx context.Context, id int64, input types.UpdateItemRequest) (types.Item, error)
	Delete(ctx context.Context, id int64) error
}

type itemService struct {
	queries sqlcgen.Querier
}

var _ ItemService = (*itemService)(nil)

func NewItemService(queries sqlcgen.Querier) ItemService {
	return &itemService{queries: queries}
}

func (s *itemService) Create(ctx context.Context, input types.CreateItemRequest) (types.Item, error) {
	if err := ctx.Err(); err != nil {
		return types.Item{}, err
	}

	input, err := normalizeItemInput(input)
	if err != nil {
		return types.Item{}, err
	}
	if err := s.validateCategory(ctx, input.CategoryID); err != nil {
		return types.Item{}, err
	}

	exists, err := s.queries.AssetTagExists(ctx, sqlcgen.AssetTagExistsParams{
		AssetTag:   input.AssetTag,
		ExcludedID: 0,
	})
	if err != nil {
		return types.Item{}, utils.UnexpectedDatabaseError(ctx, "check item asset tag", err)
	}
	if exists {
		return types.Item{}, types.ErrAssetTagConflict
	}

	row, err := s.queries.CreateItem(ctx, sqlcgen.CreateItemParams{
		CategoryID:   input.CategoryID,
		AssetTag:     input.AssetTag,
		Name:         input.Name,
		Description:  input.Description,
		SerialNumber: nullableString(input.SerialNumber),
	})
	if err != nil {
		return types.Item{}, mapItemWriteError(ctx, "create item", err)
	}

	return itemFromRow(row), nil
}

func (s *itemService) List(ctx context.Context) ([]types.Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rows, err := s.queries.ListItems(ctx)
	if err != nil {
		return nil, utils.UnexpectedDatabaseError(ctx, "list items", err)
	}

	items := make([]types.Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, itemFromRow(row))
	}

	return items, nil
}

func (s *itemService) GetByID(ctx context.Context, id int64) (types.Item, error) {
	if err := ctx.Err(); err != nil {
		return types.Item{}, err
	}
	if err := validateItemID(id); err != nil {
		return types.Item{}, err
	}

	return s.getByID(ctx, id)
}

func (s *itemService) Update(ctx context.Context, id int64, input types.UpdateItemRequest) (types.Item, error) {
	if err := ctx.Err(); err != nil {
		return types.Item{}, err
	}
	if err := validateItemID(id); err != nil {
		return types.Item{}, err
	}

	input, err := normalizeItemInput(input)
	if err != nil {
		return types.Item{}, err
	}
	if _, err := s.getByID(ctx, id); err != nil {
		return types.Item{}, err
	}
	if err := s.validateCategory(ctx, input.CategoryID); err != nil {
		return types.Item{}, err
	}

	exists, err := s.queries.AssetTagExists(ctx, sqlcgen.AssetTagExistsParams{
		AssetTag:   input.AssetTag,
		ExcludedID: id,
	})
	if err != nil {
		return types.Item{}, utils.UnexpectedDatabaseError(ctx, "check item asset tag", err)
	}
	if exists {
		return types.Item{}, types.ErrAssetTagConflict
	}

	row, err := s.queries.UpdateItem(ctx, sqlcgen.UpdateItemParams{
		CategoryID:   input.CategoryID,
		AssetTag:     input.AssetTag,
		Name:         input.Name,
		Description:  input.Description,
		SerialNumber: nullableString(input.SerialNumber),
		ID:           id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return types.Item{}, types.ErrItemNotFound
	}
	if err != nil {
		return types.Item{}, mapItemWriteError(ctx, "update item", err)
	}

	return itemFromRow(row), nil
}

func (s *itemService) Delete(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateItemID(id); err != nil {
		return err
	}

	_, err := s.queries.DeleteItem(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return types.ErrItemNotFound
	}
	if utils.PostgresErrorHasCode(err, utils.PostgresForeignKeyViolation) {
		return types.ErrItemInUse
	}
	if err != nil {
		return utils.UnexpectedDatabaseError(ctx, "delete item", err)
	}

	return nil
}

func (s *itemService) getByID(ctx context.Context, id int64) (types.Item, error) {
	row, err := s.queries.GetItem(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return types.Item{}, types.ErrItemNotFound
	}
	if err != nil {
		return types.Item{}, utils.UnexpectedDatabaseError(ctx, "get item", err)
	}

	return itemFromRow(row), nil
}

func (s *itemService) validateCategory(ctx context.Context, categoryID *int64) error {
	if categoryID == nil {
		return nil
	}
	if !utils.IsValidID(*categoryID) {
		return types.ErrInvalidCategoryID
	}

	exists, err := s.queries.CategoryExists(ctx, *categoryID)
	if err != nil {
		return utils.UnexpectedDatabaseError(ctx, "check category", err)
	}
	if !exists {
		return types.ErrCategoryNotFound
	}

	return nil
}

func itemFromRow(row sqlcgen.Item) types.Item {
	serialNumber := ""
	if row.SerialNumber != nil {
		serialNumber = *row.SerialNumber
	}

	return types.Item{
		ID:           row.ID,
		CategoryID:   row.CategoryID,
		AssetTag:     row.AssetTag,
		Name:         row.Name,
		Description:  row.Description,
		SerialNumber: serialNumber,
		Status:       row.Status,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}
}

func mapItemWriteError(ctx context.Context, operation string, err error) error {
	pgError, ok := utils.PostgresError(err)
	if !ok {
		return utils.UnexpectedDatabaseError(ctx, operation, err)
	}

	switch {
	case pgError.Code == utils.PostgresUniqueViolation && pgError.ConstraintName == constraintItemsAssetTag:
		return types.ErrAssetTagConflict
	case pgError.Code == utils.PostgresUniqueViolation && pgError.ConstraintName == constraintItemsSerial:
		return types.ErrSerialNumberConflict
	case pgError.Code == utils.PostgresForeignKeyViolation:
		return types.ErrCategoryNotFound
	default:
		return utils.UnexpectedDatabaseError(ctx, operation, err)
	}
}

func normalizeItemInput(input types.CreateItemRequest) (types.CreateItemRequest, error) {
	input.AssetTag = strings.TrimSpace(input.AssetTag)
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.SerialNumber = strings.TrimSpace(input.SerialNumber)

	if input.AssetTag == "" || input.Name == "" {
		return types.CreateItemRequest{}, types.ErrInvalidInput
	}

	return input, nil
}

func validateItemID(id int64) error {
	if !utils.IsValidID(id) {
		return types.ErrInvalidInput
	}

	return nil
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}
