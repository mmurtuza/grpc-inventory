package main
 
import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"
 
	pb "github.com/murtuza/grpc-inventory/proto/inventory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)
 
// restInventoryItem maps the JSON shape returned by the REST API.
type restInventoryItem struct {
	SKU                  string `json:"sku"`
	MedicineName         string `json:"medicine_name"`
	Quantity             int    `json:"quantity"`
	RequiresPrescription bool   `json:"requires_prescription"`
}
 
// server implements the InventoryServiceServer interface.
// Embedding UnimplementedInventoryServiceServer ensures forward compatibility —
// new RPC methods added to the .proto will not break existing server binaries.
type server struct {
	pb.UnimplementedInventoryServiceServer
	httpClient *http.Client
	apiBaseURL string
}
 
// newServer constructs a server with a shared HTTP client.
// The REST API base URL is read from the REST_API_URL environment variable,
// defaulting to localhost for local development.
func newServer() *server {
	baseURL := os.Getenv("REST_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return &server{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		apiBaseURL: baseURL,
	}
}
 
// fetchItem calls GET /api/inventory/{sku} on the REST API and returns the parsed item.
// It maps HTTP 404 to a gRPC NotFound status so the gRPC client receives a
// well-typed error rather than a raw HTTP error code.
func (s *server) fetchItem(ctx context.Context, sku string) (*restInventoryItem, error) {
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
 
	if resp.StatusCode == http.StatusNotFound {
		return nil, status.Errorf(codes.NotFound, "SKU %s not found in inventory", sku)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, status.Errorf(codes.Internal, "REST API returned unexpected status %d", resp.StatusCode)
	}
 
	var item restInventoryItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to decode REST response: %v", err)
	}
	return &item, nil
}
 
// fetchLowStock calls GET /api/inventory/low-stock and returns all matching items.
func (s *server) fetchLowStock(ctx context.Context) ([]restInventoryItem, error) {
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
 
// CheckStock is a Unary RPC. It fetches a single SKU from the REST API and
// returns the result as a typed Protobuf message.
func (s *server) CheckStock(ctx context.Context, req *pb.StockRequest) (*pb.StockResponse, error) {
	item, err := s.fetchItem(ctx, req.GetSku())
	if err != nil {
		return nil, err // already a gRPC status error
	}
	return &pb.StockResponse{
		Sku:                  item.SKU,
		MedicineName:         item.MedicineName,
		Quantity:             int32(item.Quantity),
		RequiresPrescription: item.RequiresPrescription,
	}, nil
}
 
// StreamLowStock is a Server-Streaming RPC. It fetches all low-stock items from
// the REST API in a single call, then streams each one to the client individually.
func (s *server) StreamLowStock(_ *pb.EmptyRequest, stream pb.InventoryService_StreamLowStockServer) error {
	items, err := s.fetchLowStock(stream.Context())
	if err != nil {
		return err
	}
 
	for _, item := range items {
		// Respect client cancellation between sends.
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
 
// loggingInterceptor is Unary middleware that records the method, duration,
// and outcome of every incoming RPC call.
func loggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	h, err := handler(ctx, req)
	log.Printf("[RPC] method=%s duration=%s err=%v", info.FullMethod, time.Since(start), err)
	return h, err
}
 
func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
 
	// ChainUnaryInterceptor composes multiple interceptors cleanly.
	// Add authInterceptor, recoveryInterceptor, etc. here as the service grows.
	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(loggingInterceptor),
	)
 
	pb.RegisterInventoryServiceServer(s, newServer())
 
	// Enable server reflection for debugging binary traffic
	reflection.Register(s)
 
	log.Printf("Inventory gRPC server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
