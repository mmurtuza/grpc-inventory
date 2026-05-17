-- name: GetInventoryItem :one
SELECT * FROM inventory WHERE sku = $1;

-- name: ListLowStockItems :many
SELECT * FROM inventory WHERE quantity < $1 ORDER BY quantity ASC;

-- name: CreateInventoryItem :one
INSERT INTO inventory (sku, medicine_name, quantity, requires_prescription)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateInventoryItem :one
UPDATE inventory
SET medicine_name         = $2,
    quantity              = $3,
    requires_prescription = $4,
    updated_at            = NOW()
WHERE sku = $1
RETURNING *;

-- name: DeleteInventoryItem :exec
DELETE FROM inventory WHERE sku = $1;

-- name: InventoryItemExists :one
SELECT EXISTS (SELECT 1 FROM inventory WHERE sku = $1);
