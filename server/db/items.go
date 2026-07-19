package db

import (
	"sync"

	"github.com/0xKa/equipment-checkout-system/server/types"
)

func New() *ItemsTable {
	return &ItemsTable{
		items:      make([]types.Item, 0),
		nextItemID: 1,
	}
}

type ItemsTable struct {
	mu         sync.RWMutex // protects the items slice and nextItemID
	items      []types.Item // act as an items table in a database
	nextItemID int64
}

func (table *ItemsTable) CreateItem(item types.Item) (types.Item, error) {
	table.mu.Lock()
	defer table.mu.Unlock()

	if table.assetTagExists(item.AssetTag, 0) {
		return types.Item{}, types.ErrAssetTagConflict
	}

	item.ID = table.nextItemID
	table.nextItemID++
	table.items = append(table.items, item)

	return item, nil
}

func (table *ItemsTable) ListItems() []types.Item {
	table.mu.RLock()
	defer table.mu.RUnlock()

	// Return a copy of the items slice to prevent external modification
	items := make([]types.Item, len(table.items))
	copy(items, table.items)
	return items
}

func (table *ItemsTable) GetItemByID(id int64) (types.Item, error) {
	table.mu.RLock()
	defer table.mu.RUnlock()

	index := table.itemIndex(id)
	if index == -1 {
		return types.Item{}, types.ErrItemNotFound
	}
	return table.items[index], nil
}

func (table *ItemsTable) UpdateItem(id int64, updatedItem types.Item) (types.Item, error) {
	table.mu.Lock()
	defer table.mu.Unlock()

	index := table.itemIndex(id)
	if index == -1 {
		return types.Item{}, types.ErrItemNotFound
	}

	if table.assetTagExists(updatedItem.AssetTag, id) {
		return types.Item{}, types.ErrAssetTagConflict
	}

	updatedItem.ID = id
	table.items[index] = updatedItem

	return updatedItem, nil
}

func (table *ItemsTable) DeleteItem(id int64) error {
	table.mu.Lock()
	defer table.mu.Unlock()

	index := table.itemIndex(id)
	if index == -1 {
		return types.ErrItemNotFound
	}

	// Remove the item from the slice by creating a new slice without the item
	table.items = append(table.items[:index], table.items[index+1:]...) // we do this to avoid leaving a nil entry in the slice, which could cause issues later
	return nil
}

func (table *ItemsTable) itemIndex(id int64) int {
	for index, item := range table.items {
		if item.ID == id {
			return index
		}
	}
	return -1
}

func (table *ItemsTable) assetTagExists(assetTag string, excludedID int64) bool {
	for _, item := range table.items {
		if item.AssetTag == assetTag && item.ID != excludedID {
			return true
		}
	}
	return false
}
