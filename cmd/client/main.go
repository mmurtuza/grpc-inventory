package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	pb "github.com/murtuza/grpc-inventory/proto/inventory"
	"github.com/murtuza/grpc-inventory/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

func main() {
	cfg := config.Load()
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	kaParams := keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             5 * time.Second,
		PermitWithoutStream: true,
	}

	conn, err := grpc.NewClient(
		cfg.ServerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(kaParams),
	)
	if err != nil {
		slog.Error("could not create client", "err", err)
		os.Exit(1)
	}
	defer conn.Close()

	c := pb.NewInventoryServiceClient(conn)
	runUnaryDemo(c)
	runStreamingDemo(c)
}

func runUnaryDemo(c pb.InventoryServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	slog.Info("--- Executing Unary RPC ---")
	r, err := c.CheckStock(ctx, &pb.StockRequest{Sku: "MED-001"})
	if err != nil {
		slog.Error("CheckStock failed", "err", err)
		os.Exit(1)
	}
	slog.Info("stock check result",
		"medicine", r.GetMedicineName(),
		"quantity", r.GetQuantity(),
		"prescription_required", r.GetRequiresPrescription(),
	)
}

func runStreamingDemo(c pb.InventoryServiceClient) {
	slog.Info("--- Executing Server-Streaming RPC ---")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := c.StreamLowStock(ctx, &pb.EmptyRequest{})
	if err != nil {
		slog.Error("StreamLowStock failed", "err", err)
		os.Exit(1)
	}
	for {
		item, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Error("error receiving from stream", "err", err)
			os.Exit(1)
		}
		slog.Info("low stock alert",
			"medicine", item.GetMedicineName(),
			"sku", item.GetSku(),
			"quantity", item.GetQuantity(),
		)
	}
	slog.Info("--- Stream complete ---")
}
