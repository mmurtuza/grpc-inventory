package grpcserver

import (
	"time"

	pb "github.com/murtuza/grpc-inventory/proto/inventory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// New creates a fully configured *grpc.Server with:
//   - Keepalive parameters to detect dead connections
//   - RecoveryInterceptor (outermost) → LoggingInterceptor
//   - StreamLoggingInterceptor for streaming RPCs
//   - InventoryService registered
//   - Server reflection enabled (for grpcurl debugging)
func New(svc *Service) *grpc.Server {
	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle: 30 * time.Second,
		Time:              10 * time.Second,
		Timeout:           5 * time.Second,
	}

	s := grpc.NewServer(
		grpc.KeepaliveParams(kaParams),
		// RecoveryInterceptor must be outermost so it catches panics from all inner ones.
		grpc.ChainUnaryInterceptor(RecoveryInterceptor, LoggingInterceptor),
		grpc.ChainStreamInterceptor(StreamLoggingInterceptor),
	)

	pb.RegisterInventoryServiceServer(s, svc)
	reflection.Register(s)

	return s
}
