// Package grpcserver implements the gRPC InventoryService and server wiring.
package grpcserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	pb "github.com/murtuza/grpc-inventory/proto/inventory"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// restInventoryItem maps the JSON shape returned by the REST API.
type restInventoryItem struct {
	SKU                  string `json:"sku"`
	MedicineName         string `json:"medicine_name"`
	Quantity             int    `json:"quantity"`
	RequiresPrescription bool   `json:"requires_prescription"`
}

// Service implements pb.InventoryServiceServer by proxying to the REST API.
// Embedding UnimplementedInventoryServiceServer ensures forward compatibility
// when new RPCs are added to the proto definition.
type Service struct {
	pb.UnimplementedInventoryServiceServer
	httpClient *http.Client
	apiBaseURL string
}

// NewService creates a Service that proxies requests to the given REST API base URL.
func NewService(apiBaseURL string) *Service {
	return &Service{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		apiBaseURL: apiBaseURL,
	}
}

// fetchItem calls GET /api/inventory/{sku} on the REST API.
func (s *Service) fetchItem(ctx context.Context, sku string) (*restInventoryItem, error) {
	url := fmt.Sprintf("%s/api/inventory/%s", s.apiBaseURL, sku)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to build request: %v", err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "REST API unreachable: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, status.Errorf(codes.NotFound, "SKU %s not found in inventory", sku)
	case http.StatusOK:
		// continue
	default:
		return nil, status.Errorf(codes.Internal, "REST API returned unexpected status %d", resp.StatusCode)
	}

	var item restInventoryItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to decode REST response: %v", err)
	}
	return &item, nil
}

// fetchLowStock calls GET /api/inventory/low-stock on the REST API.
func (s *Service) fetchLowStock(ctx context.Context) ([]restInventoryItem, error) {
	url := fmt.Sprintf("%s/api/inventory/low-stock", s.apiBaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to build request: %v", err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "REST API unreachable: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, status.Errorf(codes.Internal, "REST API returned unexpected status %d", resp.StatusCode)
	}
	var items []restInventoryItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to decode REST response: %v", err)
	}
	return items, nil
}

// CheckStock is a Unary RPC — fetches a single SKU from the REST API.
func (s *Service) CheckStock(ctx context.Context, req *pb.StockRequest) (*pb.StockResponse, error) {
	if req.GetSku() == "" {
		return nil, status.Error(codes.InvalidArgument, "sku is required")
	}
	item, err := s.fetchItem(ctx, req.GetSku())
	if err != nil {
		return nil, err
	}
	return &pb.StockResponse{
		Sku:                  item.SKU,
		MedicineName:         item.MedicineName,
		Quantity:             int32(item.Quantity),
		RequiresPrescription: item.RequiresPrescription,
	}, nil
}

// StreamLowStock is a Server-Streaming RPC — streams all low-stock items.
func (s *Service) StreamLowStock(_ *pb.EmptyRequest, stream pb.InventoryService_StreamLowStockServer) error {
	items, err := s.fetchLowStock(stream.Context())
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := stream.Context().Err(); err != nil {
			return status.Errorf(codes.Canceled, "client disconnected: %v", err)
		}
		if err := stream.Send(&pb.StockResponse{
			Sku:                  item.SKU,
			MedicineName:         item.MedicineName,
			Quantity:             int32(item.Quantity),
			RequiresPrescription: item.RequiresPrescription,
		}); err != nil {
			return err
		}
	}
	return nil
}
