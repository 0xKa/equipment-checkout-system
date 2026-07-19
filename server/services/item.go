package services

import (
	"context"
	"strings"
	"time"

	"github.com/0xKa/equipment-checkout-system/server/db"
	"github.com/0xKa/equipment-checkout-system/server/types"
)

type ItemService interface {
	Create(ctx context.Context, input types.CreateItemRequest) (types.Item, error)
	List(ctx context.Context) ([]types.Item, error)
	GetByID(ctx context.Context, id int64) (types.Item, error)
	Update(ctx context.Context, id int64, input types.UpdateItemRequest) (types.Item, error)
	Delete(ctx context.Context, id int64) error
}

type itemService struct {
	itemsTable *db.ItemsTable
}

var _ ItemService = (*itemService)(nil)

func NewItemService(itemsTable *db.ItemsTable) ItemService {
	return &itemService{itemsTable: itemsTable}
}

func (s *itemService) Create(ctx context.Context, input types.CreateItemRequest) (types.Item, error) {
	if err := ctx.Err(); err != nil {
		return types.Item{}, err
	}

	input, err := normalizeItemInput(input)
	if err != nil {
		return types.Item{}, err
	}

	now := time.Now().UTC()
	return s.itemsTable.CreateItem(types.Item{
		AssetTag:     input.AssetTag,
		Name:         input.Name,
		Description:  input.Description,
		SerialNumber: input.SerialNumber,
		Status:       types.ItemStatusAvailable,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
}

func (s *itemService) List(ctx context.Context) ([]types.Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return s.itemsTable.ListItems(), nil
}

func (s *itemService) GetByID(ctx context.Context, id int64) (types.Item, error) {
	if err := ctx.Err(); err != nil {
		return types.Item{}, err
	}
	if err := validateItemID(id); err != nil {
		return types.Item{}, err
	}

	return s.itemsTable.GetItemByID(id)
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

	item, err := s.itemsTable.GetItemByID(id)
	if err != nil {
		return types.Item{}, err
	}

	updatedAt := time.Now().UTC()
	if !updatedAt.After(item.UpdatedAt) {
		updatedAt = item.UpdatedAt.Add(time.Nanosecond)
	}

	// PUT replaces client-editable fields and preserves all server-owned values.
	item.AssetTag = input.AssetTag
	item.Name = input.Name
	item.Description = input.Description
	item.SerialNumber = input.SerialNumber
	item.UpdatedAt = updatedAt

	return s.itemsTable.UpdateItem(id, item)
}

func (s *itemService) Delete(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateItemID(id); err != nil {
		return err
	}

	return s.itemsTable.DeleteItem(id)
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
	if id <= 0 {
		return types.ErrInvalidInput
	}

	return nil
}
