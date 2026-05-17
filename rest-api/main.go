package main
 
import (
	"encoding/json"
	"log"
	"net/http"
)
 
// StockItem mirrors the shape we will return from the REST API.
// The gRPC server will map these fields onto its Protobuf StockResponse.
type StockItem struct {
	SKU                  string `json:"sku"`
	MedicineName         string `json:"medicine_name"`
	Quantity             int    `json:"quantity"`
	RequiresPrescription bool   `json:"requires_prescription"`
}
 
// inventory is the in-memory data store for this service.
// In production this would be replaced by a database query.
var inventory = map[string]StockItem{
	"MED-001": {SKU: "MED-001", MedicineName: "Amoxicillin 500mg", Quantity: 1450, RequiresPrescription: true},
	"MED-045": {SKU: "MED-045", MedicineName: "Metformin 500mg", Quantity: 320, RequiresPrescription: true},
	"MED-089": {SKU: "MED-089", MedicineName: "Ibuprofen 200mg", Quantity: 12, RequiresPrescription: false},
	"MED-102": {SKU: "MED-102", MedicineName: "Lisinopril 10mg", Quantity: 5, RequiresPrescription: true},
	"MED-201": {SKU: "MED-201", MedicineName: "Cetirizine 10mg", Quantity: 8, RequiresPrescription: false},
}
 
// lowStockThreshold defines what "low stock" means across the system.
const lowStockThreshold = 50
 
// writeJSON is a small helper that sets the Content-Type header and encodes
// the provided value as JSON. It writes a 500 if encoding fails.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, `{"error":"encoding failed"}`, http.StatusInternalServerError)
	}
}
 
// getStock handles GET /api/inventory/{sku}.
// It returns the stock record for a single SKU or 404 if not found.
func getStock(w http.ResponseWriter, r *http.Request) {
	sku := r.PathValue("sku")
	item, ok := inventory[sku]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "SKU not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}
 
// getLowStock handles GET /api/inventory/low-stock.
// It returns all items whose quantity falls below lowStockThreshold.
func getLowStock(w http.ResponseWriter, r *http.Request) {
	var items []StockItem
	for _, item := range inventory {
		if item.Quantity < lowStockThreshold {
			items = append(items, item)
		}
	}
	writeJSON(w, http.StatusOK, items)
}
 
func main() {
	mux := http.NewServeMux()
 
	// Register the specific /low-stock route BEFORE the wildcard {sku} route.
	// Go 1.22+ ServeMux resolves conflicts by specificity, but explicit ordering
	// makes the intent clear to future maintainers.
	mux.HandleFunc("GET /api/inventory/low-stock", getLowStock)
	mux.HandleFunc("GET /api/inventory/{sku}", getStock)
 
	log.Println("REST Inventory API listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
