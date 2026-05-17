package grpcserver

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor records the method, duration, and outcome of every unary RPC.
func LoggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	h, err := handler(ctx, req)
	slog.Info("unary rpc",
		"method", info.FullMethod,
		"duration", time.Since(start).String(),
		"err", err,
	)
	return h, err
}

// RecoveryInterceptor catches panics in RPC handlers and converts them to
// codes.Internal, preventing a single bad request from crashing the server.
func RecoveryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic recovered in rpc handler",
				"method", info.FullMethod,
				"panic", r,
				"stack", string(debug.Stack()),
			)
			err = status.Errorf(codes.Internal, "internal server error")
		}
	}()
	return handler(ctx, req)
}

// StreamLoggingInterceptor mirrors LoggingInterceptor for server-streaming RPCs.
func StreamLoggingInterceptor(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	start := time.Now()
	err := handler(srv, ss)
	slog.Info("stream rpc",
		"method", info.FullMethod,
		"duration", time.Since(start).String(),
		"err", err,
	)
	return err
}
