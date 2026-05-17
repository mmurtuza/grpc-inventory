package grpcserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pb "github.com/murtuza/grpc-inventory/proto/inventory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ─── fetchItem ────────────────────────────────────────────────────────────────

func TestFetchItem_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(restInventoryItem{SKU: "MED-001", MedicineName: "Amoxicillin", Quantity: 100, RequiresPrescription: true})
	}))
	defer ts.Close()

	svc := &Service{httpClient: ts.Client(), apiBaseURL: ts.URL}
	item, err := svc.fetchItem(context.Background(), "MED-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.SKU != "MED-001" {
		t.Errorf("expected MED-001, got %q", item.SKU)
	}
}

func TestFetchItem_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	svc := &Service{httpClient: ts.Client(), apiBaseURL: ts.URL}
	_, err := svc.fetchItem(context.Background(), "X")
	if code := status.Code(err); code != codes.NotFound {
		t.Errorf("expected NotFound, got %v", code)
	}
}

func TestFetchItem_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()
	svc := &Service{httpClient: ts.Client(), apiBaseURL: ts.URL}
	_, err := svc.fetchItem(context.Background(), "X")
	if code := status.Code(err); code != codes.Internal {
		t.Errorf("expected Internal, got %v", code)
	}
}

func TestFetchItem_Unreachable(t *testing.T) {
	svc := &Service{httpClient: &http.Client{Timeout: 200 * time.Millisecond}, apiBaseURL: "http://127.0.0.1:19999"}
	_, err := svc.fetchItem(context.Background(), "X")
	if code := status.Code(err); code != codes.Unavailable {
		t.Errorf("expected Unavailable, got %v", code)
	}
}

func TestFetchItem_BadJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("not-json"))
	}))
	defer ts.Close()
	svc := &Service{httpClient: ts.Client(), apiBaseURL: ts.URL}
	_, err := svc.fetchItem(context.Background(), "X")
	if code := status.Code(err); code != codes.Internal {
		t.Errorf("expected Internal, got %v", code)
	}
}

// ─── fetchLowStock ────────────────────────────────────────────────────────────

func TestFetchLowStock_Success(t *testing.T) {
	items := []restInventoryItem{{SKU: "A", Quantity: 5}, {SKU: "B", Quantity: 3}}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(items)
	}))
	defer ts.Close()
	svc := &Service{httpClient: ts.Client(), apiBaseURL: ts.URL}
	got, err := svc.fetchLowStock(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}
}

func TestFetchLowStock_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()
	svc := &Service{httpClient: ts.Client(), apiBaseURL: ts.URL}
	_, err := svc.fetchLowStock(context.Background())
	if code := status.Code(err); code != codes.Internal {
		t.Errorf("expected Internal, got %v", code)
	}
}

// ─── CheckStock ───────────────────────────────────────────────────────────────

func TestCheckStock_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(restInventoryItem{SKU: "MED-001", MedicineName: "Amoxicillin", Quantity: 100, RequiresPrescription: true})
	}))
	defer ts.Close()
	svc := &Service{httpClient: ts.Client(), apiBaseURL: ts.URL}
	resp, err := svc.CheckStock(context.Background(), &pb.StockRequest{Sku: "MED-001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetSku() != "MED-001" {
		t.Errorf("expected MED-001, got %q", resp.GetSku())
	}
	if resp.GetQuantity() != 100 {
		t.Errorf("expected 100, got %d", resp.GetQuantity())
	}
}

func TestCheckStock_EmptySKU(t *testing.T) {
	svc := &Service{httpClient: &http.Client{}, apiBaseURL: "http://localhost"}
	_, err := svc.CheckStock(context.Background(), &pb.StockRequest{Sku: ""})
	if code := status.Code(err); code != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", code)
	}
}

func TestCheckStock_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	svc := &Service{httpClient: ts.Client(), apiBaseURL: ts.URL}
	_, err := svc.CheckStock(context.Background(), &pb.StockRequest{Sku: "MED-999"})
	if code := status.Code(err); code != codes.NotFound {
		t.Errorf("expected NotFound, got %v", code)
	}
}

// ─── Interceptors ─────────────────────────────────────────────────────────────

func TestLoggingInterceptor(t *testing.T) {
	called := false
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		called = true
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	resp, err := LoggingInterceptor(context.Background(), nil, info, handler)
	if err != nil || !called || resp != "ok" {
		t.Errorf("unexpected result: resp=%v err=%v called=%v", resp, err, called)
	}
}

func TestRecoveryInterceptor_CatchesPanic(t *testing.T) {
	handler := func(_ context.Context, _ interface{}) (interface{}, error) { panic("boom") }
	info := &grpc.UnaryServerInfo{FullMethod: "/test/Panic"}
	_, err := RecoveryInterceptor(context.Background(), nil, info, handler)
	if code := status.Code(err); code != codes.Internal {
		t.Errorf("expected Internal, got %v", code)
	}
}

func TestRecoveryInterceptor_PassesThrough(t *testing.T) {
	handler := func(_ context.Context, _ interface{}) (interface{}, error) { return "ok", nil }
	info := &grpc.UnaryServerInfo{FullMethod: "/test/Normal"}
	resp, err := RecoveryInterceptor(context.Background(), nil, info, handler)
	if err != nil || resp != "ok" {
		t.Errorf("unexpected: resp=%v err=%v", resp, err)
	}
}
