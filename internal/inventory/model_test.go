package inventory

import (
	"testing"
)

func TestValidateStockItem(t *testing.T) {
	tests := []struct {
		name    string
		item    StockItem
		wantErr bool
	}{
		{"valid", StockItem{SKU: "X", MedicineName: "Y", Quantity: 0}, false},
		{"empty sku", StockItem{SKU: "", MedicineName: "Y", Quantity: 0}, true},
		{"whitespace sku", StockItem{SKU: "  ", MedicineName: "Y", Quantity: 0}, true},
		{"empty name", StockItem{SKU: "X", MedicineName: "", Quantity: 0}, true},
		{"negative qty", StockItem{SKU: "X", MedicineName: "Y", Quantity: -1}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateStockItem(tc.item)
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
