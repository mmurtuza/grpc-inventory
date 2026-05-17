// Package inventory contains all REST API business logic for the inventory service.
package inventory

import (
	"errors"
	"strings"
)

// StockItem is the API-facing representation of an inventory record.
type StockItem struct {
	SKU                  string `json:"sku"`
	MedicineName         string `json:"medicine_name"`
	Quantity             int    `json:"quantity"`
	RequiresPrescription bool   `json:"requires_prescription"`
}

// LowStockThreshold defines what "low stock" means across the system.
const LowStockThreshold int32 = 50

// validateStockItem enforces business rules on incoming request bodies.
func validateStockItem(item StockItem) error {
	if strings.TrimSpace(item.SKU) == "" {
		return errors.New("sku is required")
	}
	if strings.TrimSpace(item.MedicineName) == "" {
		return errors.New("medicine_name is required")
	}
	if item.Quantity < 0 {
		return errors.New("quantity must be non-negative")
	}
	return nil
}
