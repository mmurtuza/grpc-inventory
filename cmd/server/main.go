package main

import (
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/murtuza/grpc-inventory/internal/config"
	"github.com/murtuza/grpc-inventory/internal/grpcserver"
)

func main() {
	cfg := config.Load()

	if cfg.LogFormat == "json" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	}

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		slog.Error("failed to listen", "err", err)
		os.Exit(1)
	}

	svc := grpcserver.NewService(cfg.RestAPIURL)
	srv := grpcserver.New(svc)

	go func() {
		slog.Info("Inventory gRPC server listening", "addr", lis.Addr())
		if err := srv.Serve(lis); err != nil {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down gRPC server gracefully...")
	srv.GracefulStop()
	slog.Info("gRPC server stopped")
}
