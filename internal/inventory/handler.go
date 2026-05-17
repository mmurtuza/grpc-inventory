package inventory

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/murtuza/grpc-inventory/internal/cache"
)

// Pinger is a minimal interface satisfied by *pgxpool.Pool and *cache.RedisCache.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Handler holds all dependencies injected at startup.
type Handler struct {
	store    Store
	cache    cache.Cache
	dbPinger Pinger
}

// NewHandler creates a Handler with the given dependencies.
func NewHandler(store Store, c cache.Cache, dbPinger Pinger) *Handler {
	return &Handler{store: store, cache: c, dbPinger: dbPinger}
}

// Healthz handles GET /healthz — pings both PostgreSQL and Redis.
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	dbStatus, cacheStatus := "ok", "ok"

	if h.dbPinger != nil {
		if err := h.dbPinger.Ping(ctx); err != nil {
			dbStatus = "error"
			slog.Error("healthz: db ping failed", "err", err)
		}
	}
	if err := h.cache.Ping(ctx); err != nil {
		cacheStatus = "error"
		slog.Error("healthz: cache ping failed", "err", err)
	}

	overall, httpStatus := "ok", http.StatusOK
	if dbStatus != "ok" || cacheStatus != "ok" {
		overall, httpStatus = "degraded", http.StatusServiceUnavailable
	}
	writeJSON(w, httpStatus, map[string]string{"status": overall, "db": dbStatus, "cache": cacheStatus})
}

// GetStock handles GET /api/inventory/{sku} — cache-aside read.
func (h *Handler) GetStock(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sku := r.PathValue("sku")
	key := cache.SKUKey(sku)

	if raw, hit, _ := h.cache.Get(ctx, key); hit {
		var item StockItem
		if json.Unmarshal([]byte(raw), &item) == nil {
			slog.Debug("cache hit", "key", key)
			writeJSON(w, http.StatusOK, item)
			return
		}
	}

	item, err := h.store.GetInventoryItem(ctx, sku)
	if errors.Is(err, ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "SKU not found"})
		return
	}
	if err != nil {
		slog.Error("GetStock: db error", "sku", sku, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	setCache(h.cache, ctx, key, item, cache.SKUTTL)
	writeJSON(w, http.StatusOK, item)
}

// GetLowStock handles GET /api/inventory/low-stock — cache-aside read.
func (h *Handler) GetLowStock(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := cache.LowStockKey()

	if raw, hit, _ := h.cache.Get(ctx, key); hit {
		var items []StockItem
		if json.Unmarshal([]byte(raw), &items) == nil {
			slog.Debug("cache hit", "key", key)
			writeJSON(w, http.StatusOK, items)
			return
		}
	}

	items, err := h.store.ListLowStockItems(ctx, LowStockThreshold)
	if err != nil {
		slog.Error("GetLowStock: db error", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	setCache(h.cache, ctx, key, items, cache.LowStockTTL)
	writeJSON(w, http.StatusOK, items)
}

// AddStock handles POST /api/inventory.
func (h *Handler) AddStock(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var item StockItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if err := validateStockItem(item); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	exists, err := h.store.InventoryItemExists(ctx, item.SKU)
	if err != nil {
		slog.Error("AddStock: exists check failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if exists {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "SKU already exists"})
		return
	}
	created, err := h.store.CreateInventoryItem(ctx, item)
	if err != nil {
		slog.Error("AddStock: create failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	_ = h.cache.Delete(ctx, cache.LowStockKey())
	writeJSON(w, http.StatusCreated, created)
}

// RemoveStock handles DELETE /api/inventory/{sku}.
func (h *Handler) RemoveStock(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sku := r.PathValue("sku")
	exists, err := h.store.InventoryItemExists(ctx, sku)
	if err != nil {
		slog.Error("RemoveStock: exists check failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "SKU not found"})
		return
	}
	if err := h.store.DeleteInventoryItem(ctx, sku); err != nil {
		slog.Error("RemoveStock: delete failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	_ = h.cache.Delete(ctx, cache.SKUKey(sku), cache.LowStockKey())
	writeJSON(w, http.StatusOK, map[string]string{"message": "stock removed successfully"})
}

// UpdateStock handles PUT /api/inventory/{sku}.
func (h *Handler) UpdateStock(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sku := r.PathValue("sku")
	exists, err := h.store.InventoryItemExists(ctx, sku)
	if err != nil {
		slog.Error("UpdateStock: exists check failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "SKU not found"})
		return
	}
	var item StockItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if err := validateStockItem(item); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	item.SKU = sku // path SKU always wins over body
	updated, err := h.store.UpdateInventoryItem(ctx, item)
	if err != nil {
		slog.Error("UpdateStock: update failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	_ = h.cache.Delete(ctx, cache.SKUKey(sku), cache.LowStockKey())
	writeJSON(w, http.StatusOK, updated)
}

// ─── private helpers ─────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode JSON response", "err", err)
	}
}

// setCache marshals v and stores it in the cache. Errors are logged, never returned.
func setCache(c cache.Cache, ctx context.Context, key string, v any, ttl time.Duration) {
	b, err := json.Marshal(v)
	if err != nil {
		slog.Warn("cache marshal failed", "key", key, "err", err)
		return
	}
	if err := c.Set(ctx, key, string(b), ttl); err != nil {
		slog.Warn("cache set failed", "key", key, "err", err)
	}
}
