package services

import (
	"context"

	"github.com/0xKa/equipment-checkout-system/server/types"
)

type ItemService interface {
	Create(ctx context.Context, input types.CreateItemRequest) (types.Item, error)
	List(ctx context.Context) ([]types.Item, error)
	GetByID(ctx context.Context, id int64) (types.Item, error)
	Update(ctx context.Context, id int64, input types.UpdateItemRequest) (types.Item, error)
	Delete(ctx context.Context, id int64) error
}
