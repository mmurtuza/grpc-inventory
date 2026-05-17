package inventory

import (
	"context"
	"errors"

	db "github.com/murtuza/grpc-inventory/db/sqlc"
	"github.com/jackc/pgx/v5"
)

// ErrNotFound is returned by Store when a requested SKU does not exist.
var ErrNotFound = errors.New("inventory item not found")

// Store is the persistence interface used by Handler.
// pgStore provides the real PostgreSQL implementation; tests use a mock.
type Store interface {
	GetInventoryItem(ctx context.Context, sku string) (StockItem, error)
	ListLowStockItems(ctx context.Context, threshold int32) ([]StockItem, error)
	CreateInventoryItem(ctx context.Context, item StockItem) (StockItem, error)
	UpdateInventoryItem(ctx context.Context, item StockItem) (StockItem, error)
	DeleteInventoryItem(ctx context.Context, sku string) error
	InventoryItemExists(ctx context.Context, sku string) (bool, error)
}

// NewPgStore creates a Store backed by PostgreSQL via sqlc.
func NewPgStore(q *db.Queries) Store { return &pgStore{q: q} }

type pgStore struct{ q *db.Queries }

func toStockItem(row db.Inventory) StockItem {
	return StockItem{
		SKU:                  row.Sku,
		MedicineName:         row.MedicineName,
		Quantity:             int(row.Quantity),
		RequiresPrescription: row.RequiresPrescription,
	}
}

func (s *pgStore) GetInventoryItem(ctx context.Context, sku string) (StockItem, error) {
	row, err := s.q.GetInventoryItem(ctx, sku)
	if errors.Is(err, pgx.ErrNoRows) {
		return StockItem{}, ErrNotFound
	}
	if err != nil {
		return StockItem{}, err
	}
	return toStockItem(row), nil
}

func (s *pgStore) ListLowStockItems(ctx context.Context, threshold int32) ([]StockItem, error) {
	rows, err := s.q.ListLowStockItems(ctx, threshold)
	if err != nil {
		return nil, err
	}
	items := make([]StockItem, len(rows))
	for i, row := range rows {
		items[i] = toStockItem(row)
	}
	return items, nil
}

func (s *pgStore) CreateInventoryItem(ctx context.Context, item StockItem) (StockItem, error) {
	row, err := s.q.CreateInventoryItem(ctx, db.CreateInventoryItemParams{
		Sku:                  item.SKU,
		MedicineName:         item.MedicineName,
		Quantity:             int32(item.Quantity),
		RequiresPrescription: item.RequiresPrescription,
	})
	if err != nil {
		return StockItem{}, err
	}
	return toStockItem(row), nil
}

func (s *pgStore) UpdateInventoryItem(ctx context.Context, item StockItem) (StockItem, error) {
	row, err := s.q.UpdateInventoryItem(ctx, db.UpdateInventoryItemParams{
		Sku:                  item.SKU,
		MedicineName:         item.MedicineName,
		Quantity:             int32(item.Quantity),
		RequiresPrescription: item.RequiresPrescription,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return StockItem{}, ErrNotFound
	}
	if err != nil {
		return StockItem{}, err
	}
	return toStockItem(row), nil
}

func (s *pgStore) DeleteInventoryItem(ctx context.Context, sku string) error {
	return s.q.DeleteInventoryItem(ctx, sku)
}

func (s *pgStore) InventoryItemExists(ctx context.Context, sku string) (bool, error) {
	return s.q.InventoryItemExists(ctx, sku)
}
