package inventory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/murtuza/grpc-inventory/internal/cache"
)

// ─── Mock Store ───────────────────────────────────────────────────────────────

type mockStore struct{ items map[string]StockItem }

func newMockStore() *mockStore {
	return &mockStore{items: map[string]StockItem{
		"MED-001": {SKU: "MED-001", MedicineName: "Amoxicillin 500mg", Quantity: 1450, RequiresPrescription: true},
		"MED-045": {SKU: "MED-045", MedicineName: "Metformin 500mg", Quantity: 320, RequiresPrescription: true},
		"MED-089": {SKU: "MED-089", MedicineName: "Ibuprofen 200mg", Quantity: 12, RequiresPrescription: false},
		"MED-102": {SKU: "MED-102", MedicineName: "Lisinopril 10mg", Quantity: 5, RequiresPrescription: true},
		"MED-201": {SKU: "MED-201", MedicineName: "Cetirizine 10mg", Quantity: 8, RequiresPrescription: false},
	}}
}
func (m *mockStore) GetInventoryItem(_ context.Context, sku string) (StockItem, error) {
	if v, ok := m.items[sku]; ok {
		return v, nil
	}
	return StockItem{}, ErrNotFound
}
func (m *mockStore) ListLowStockItems(_ context.Context, threshold int32) ([]StockItem, error) {
	var out []StockItem
	for _, v := range m.items {
		if int32(v.Quantity) < threshold {
			out = append(out, v)
		}
	}
	if out == nil {
		out = []StockItem{}
	}
	return out, nil
}
func (m *mockStore) CreateInventoryItem(_ context.Context, item StockItem) (StockItem, error) {
	m.items[item.SKU] = item
	return item, nil
}
func (m *mockStore) UpdateInventoryItem(_ context.Context, item StockItem) (StockItem, error) {
	if _, ok := m.items[item.SKU]; !ok {
		return StockItem{}, ErrNotFound
	}
	m.items[item.SKU] = item
	return item, nil
}
func (m *mockStore) DeleteInventoryItem(_ context.Context, sku string) error {
	delete(m.items, sku)
	return nil
}
func (m *mockStore) InventoryItemExists(_ context.Context, sku string) (bool, error) {
	_, ok := m.items[sku]
	return ok, nil
}

// ─── Mock Cache ───────────────────────────────────────────────────────────────

type mockCache struct{ data map[string]string }

func newMockCache() *mockCache { return &mockCache{data: make(map[string]string)} }
func (m *mockCache) Get(_ context.Context, k string) (string, bool, error) {
	v, ok := m.data[k]
	return v, ok, nil
}
func (m *mockCache) Set(_ context.Context, k, v string, _ time.Duration) error {
	m.data[k] = v
	return nil
}
func (m *mockCache) Delete(_ context.Context, keys ...string) error {
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}
func (m *mockCache) Ping(_ context.Context) error { return nil }

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newTestHandler() (*Handler, *mockStore, *mockCache) {
	s, c := newMockStore(), newMockCache()
	return NewHandler(s, c, nil), s, c
}

func newTestMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.Healthz)
	mux.HandleFunc("GET /api/inventory/low-stock", h.GetLowStock)
	mux.HandleFunc("GET /api/inventory/{sku}", h.GetStock)
	mux.HandleFunc("POST /api/inventory", h.AddStock)
	mux.HandleFunc("DELETE /api/inventory/{sku}", h.RemoveStock)
	mux.HandleFunc("PUT /api/inventory/{sku}", h.UpdateStock)
	return mux
}

// ─── Healthz ──────────────────────────────────────────────────────────────────

func TestHealthz(t *testing.T) {
	h, _, _ := newTestHandler()
	w := httptest.NewRecorder()
	h.Healthz(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("expected ok, got %q", body["status"])
	}
}

// ─── GetStock ────────────────────────────────────────────────────────────────

func TestGetStock_Found(t *testing.T) {
	h, _, _ := newTestHandler()
	mux := newTestMux(h)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/inventory/MED-001", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var item StockItem
	json.NewDecoder(w.Body).Decode(&item)
	if item.SKU != "MED-001" {
		t.Errorf("expected MED-001, got %q", item.SKU)
	}
}

func TestGetStock_NotFound(t *testing.T) {
	h, _, _ := newTestHandler()
	mux := newTestMux(h)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/inventory/MED-999", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetStock_CacheHit(t *testing.T) {
	h, _, c := newTestHandler()
	cached := StockItem{SKU: "MED-001", MedicineName: "Cached", Quantity: 1}
	b, _ := json.Marshal(cached)
	c.data[cache.SKUKey("MED-001")] = string(b)

	mux := newTestMux(h)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/inventory/MED-001", nil))

	var got StockItem
	json.NewDecoder(w.Body).Decode(&got)
	if got.MedicineName != "Cached" {
		t.Errorf("expected cache hit value, got %q", got.MedicineName)
	}
}

func TestGetStock_CacheMissPopulates(t *testing.T) {
	h, _, c := newTestHandler()
	mux := newTestMux(h)
	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/inventory/MED-001", nil))
	if _, ok := c.data[cache.SKUKey("MED-001")]; !ok {
		t.Error("expected MED-001 to be cached after miss")
	}
}

// ─── GetLowStock ──────────────────────────────────────────────────────────────

func TestGetLowStock(t *testing.T) {
	h, _, _ := newTestHandler()
	w := httptest.NewRecorder()
	h.GetLowStock(w, httptest.NewRequest(http.MethodGet, "/api/inventory/low-stock", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var items []StockItem
	json.NewDecoder(w.Body).Decode(&items)
	if len(items) != 3 {
		t.Errorf("expected 3 low-stock items, got %d", len(items))
	}
}

func TestGetLowStock_CacheHit(t *testing.T) {
	h, _, c := newTestHandler()
	cached := []StockItem{{SKU: "CACHED"}}
	b, _ := json.Marshal(cached)
	c.data[cache.LowStockKey()] = string(b)
	w := httptest.NewRecorder()
	h.GetLowStock(w, httptest.NewRequest(http.MethodGet, "/api/inventory/low-stock", nil))
	var items []StockItem
	json.NewDecoder(w.Body).Decode(&items)
	if len(items) != 1 || items[0].SKU != "CACHED" {
		t.Errorf("expected cached result, got %v", items)
	}
}

// ─── AddStock ─────────────────────────────────────────────────────────────────

func TestAddStock(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"valid", `{"sku":"MED-999","medicine_name":"Test","quantity":100}`, http.StatusCreated},
		{"invalid JSON", `{bad`, http.StatusBadRequest},
		{"empty SKU", `{"sku":"","medicine_name":"Test","quantity":10}`, http.StatusUnprocessableEntity},
		{"negative qty", `{"sku":"MED-888","medicine_name":"Test","quantity":-1}`, http.StatusUnprocessableEntity},
		{"duplicate", `{"sku":"MED-001","medicine_name":"X","quantity":10}`, http.StatusConflict},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, _, _ := newTestHandler()
			w := httptest.NewRecorder()
			h.AddStock(w, httptest.NewRequest(http.MethodPost, "/api/inventory", strings.NewReader(tc.body)))
			if w.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d (body: %s)", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestAddStock_InvalidatesLowStockCache(t *testing.T) {
	h, _, c := newTestHandler()
	c.data[cache.LowStockKey()] = `[]`
	body := `{"sku":"MED-NEW","medicine_name":"New Drug","quantity":5}`
	w := httptest.NewRecorder()
	h.AddStock(w, httptest.NewRequest(http.MethodPost, "/api/inventory", strings.NewReader(body)))
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	if _, ok := c.data[cache.LowStockKey()]; ok {
		t.Error("expected low-stock cache invalidated")
	}
}

// ─── RemoveStock ──────────────────────────────────────────────────────────────

func TestRemoveStock(t *testing.T) {
	mux := newTestMux(func() *Handler { h, _, _ := newTestHandler(); return h }())
	for _, tc := range []struct {
		sku  string
		want int
	}{{"MED-001", 200}, {"MED-999", 404}} {
		h, _, _ := newTestHandler()
		mux := newTestMux(h)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/inventory/"+tc.sku, nil))
		if w.Code != tc.want {
			t.Errorf("sku=%s: expected %d, got %d", tc.sku, tc.want, w.Code)
		}
	}
	_ = mux
}

func TestRemoveStock_InvalidatesCaches(t *testing.T) {
	h, _, c := newTestHandler()
	c.data[cache.SKUKey("MED-001")] = `{}`
	c.data[cache.LowStockKey()] = `[]`
	mux := newTestMux(h)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/inventory/MED-001", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if _, ok := c.data[cache.SKUKey("MED-001")]; ok {
		t.Error("expected SKU cache invalidated")
	}
	if _, ok := c.data[cache.LowStockKey()]; ok {
		t.Error("expected low-stock cache invalidated")
	}
}

// ─── UpdateStock ──────────────────────────────────────────────────────────────

func TestUpdateStock(t *testing.T) {
	tests := []struct {
		name string
		sku  string
		body string
		want int
	}{
		{"valid", "MED-001", `{"sku":"MED-001","medicine_name":"Amox","quantity":500,"requires_prescription":true}`, 200},
		{"not found", "MED-999", `{"sku":"MED-999","medicine_name":"X","quantity":10}`, 404},
		{"bad JSON", "MED-001", `{bad`, 400},
		{"negative qty", "MED-001", `{"sku":"MED-001","medicine_name":"X","quantity":-5}`, 422},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, _, _ := newTestHandler()
			mux := newTestMux(h)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/inventory/"+tc.sku, strings.NewReader(tc.body)))
			if w.Code != tc.want {
				t.Errorf("expected %d, got %d (body: %s)", tc.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestUpdateStock_SKUNotOverridden(t *testing.T) {
	h, store, _ := newTestHandler()
	mux := newTestMux(h)
	body := `{"sku":"WRONG","medicine_name":"X","quantity":100,"requires_prescription":true}`
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/inventory/MED-001", strings.NewReader(body)))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if _, ok := store.items["MED-001"]; !ok {
		t.Error("MED-001 should still exist")
	}
}

// ─── validateStockItem ───────────────────────────────────────────────────────

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

// ─── Middleware ───────────────────────────────────────────────────────────────

func TestLoggingMiddleware(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	w := httptest.NewRecorder()
	LoggingMiddleware(inner).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusTeapot {
		t.Errorf("expected 418, got %d", w.Code)
	}
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	w := httptest.NewRecorder()
	CORSMiddleware("*")(inner).ServeHTTP(w, httptest.NewRequest(http.MethodOptions, "/", nil))
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

// ─── cache package helpers ────────────────────────────────────────────────────

func TestNoopCache(t *testing.T) {
	c := cache.NoopCache{}
	ctx := context.Background()
	if _, ok, err := c.Get(ctx, "k"); ok || err != nil {
		t.Error("NoopCache.Get should return false, nil")
	}
	if err := c.Set(ctx, "k", "v", time.Minute); err != nil {
		t.Errorf("NoopCache.Set unexpected error: %v", err)
	}
}

func TestCacheKeys(t *testing.T) {
	if got := cache.SKUKey("MED-001"); got != "inv:sku:MED-001" {
		t.Errorf("got %q", got)
	}
	if got := cache.LowStockKey(); got != "inv:low-stock" {
		t.Errorf("got %q", got)
	}
}
