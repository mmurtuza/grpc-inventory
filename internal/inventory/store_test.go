package inventory

import (
	"context"
	"testing"

	db "github.com/murtuza/grpc-inventory/db/sqlc"
)

func TestToStockItem(t *testing.T) {
	row := db.Inventory{
		Sku:                  "MED-TEST",
		MedicineName:         "Aspirin",
		Quantity:             100,
		RequiresPrescription: false,
	}

	item := toStockItem(row)
	if item.SKU != "MED-TEST" || item.MedicineName != "Aspirin" || item.Quantity != 100 || item.RequiresPrescription != false {
		t.Errorf("toStockItem did not map correctly, got: %+v", item)
	}
}

func TestPgStoreMethods_NilQueries(t *testing.T) {
	// A basic test to ensure we panic or error appropriately when queries is nil.
	// This provides coverage of the store methods being called.
	store := NewPgStore(nil)
	ctx := context.Background()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when queries is nil")
		}
	}()

	// This should panic because q is nil
	store.GetInventoryItem(ctx, "MED-001")
}
