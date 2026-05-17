package main
 
import (
	"context"
	"io"
	"log"
	"time"
 
	pb "github.com/murtuza/grpc-inventory/proto/inventory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)
 
func main() {
	// grpc.NewClient does not block — the connection is established lazily.
	// For production, replace insecure.NewCredentials() with a TLS config.
	conn, err := grpc.NewClient(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("could not create client: %v", err)
	}
	defer conn.Close()
 
	c := pb.NewInventoryServiceClient(conn)
 
	testUnaryCall(c)
	testStreamingCall(c)
}
 
func testUnaryCall(c pb.InventoryServiceClient) {
	// A deadline context is essential for unary calls. Without one, a slow or
	// unresponsive server will block the caller indefinitely.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
 
	log.Println("--- Executing Unary RPC ---")
	r, err := c.CheckStock(ctx, &pb.StockRequest{Sku: "MED-001"})
	if err != nil {
		log.Fatalf("CheckStock failed: %v", err)
	}
	log.Printf("Stock check -> %s: %d units (prescription required: %t)",
		r.GetMedicineName(), r.GetQuantity(), r.GetRequiresPrescription())
}
 
func testStreamingCall(c pb.InventoryServiceClient) {
	log.Println("\n--- Executing Server-Streaming RPC ---")
 
	// A deadline is equally important on streaming calls. Use context.WithCancel
	// if you need to abort the stream from the client side based on your own logic.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
 
	stream, err := c.StreamLowStock(ctx, &pb.EmptyRequest{})
	if err != nil {
		log.Fatalf("StreamLowStock failed: %v", err)
	}
 
	for {
		item, err := stream.Recv()
		if err == io.EOF {
			// The server closed the stream normally.
			break
		}
		if err != nil {
			log.Fatalf("error receiving from stream: %v", err)
		}
		log.Printf("ALERT: Low stock -> %s (SKU: %s) has only %d units remaining.",
			item.GetMedicineName(), item.GetSku(), item.GetQuantity())
	}
	log.Println("--- Stream complete ---")
}
