package inventory

import "net/http"

// NewRouter registers all inventory routes on a ServeMux and wraps it with
// the CORS and logging middleware chain. Returns a fully composed http.Handler.
func NewRouter(h *Handler, allowedOrigin string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", h.Healthz)
	// Register /low-stock before the /{sku} wildcard so both match correctly.
	mux.HandleFunc("GET /api/inventory/low-stock", h.GetLowStock)
	mux.HandleFunc("GET /api/inventory/{sku}", h.GetStock)
	mux.HandleFunc("POST /api/inventory", h.AddStock)
	mux.HandleFunc("DELETE /api/inventory/{sku}", h.RemoveStock)
	mux.HandleFunc("PUT /api/inventory/{sku}", h.UpdateStock)

	// Chain: CORS → Logging → Mux
	return CORSMiddleware(allowedOrigin)(LoggingMiddleware(mux))
}
