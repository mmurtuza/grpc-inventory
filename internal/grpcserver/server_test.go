package grpcserver

import (
	"context"
	"testing"

	"google.golang.org/grpc"
)

func TestNewServer(t *testing.T) {
	svc := NewService("http://localhost:8080")
	srv := New(svc)
	if srv == nil {
		t.Fatal("expected server, got nil")
	}
}

type mockServerStream struct {
	grpc.ServerStream
}

func (m *mockServerStream) Context() context.Context {
	return context.Background()
}

func TestStreamLoggingInterceptor(t *testing.T) {
	called := false
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		called = true
		return nil
	}

	info := &grpc.StreamServerInfo{FullMethod: "/test/Stream"}
	err := StreamLoggingInterceptor(nil, &mockServerStream{}, info, handler)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected handler to be called")
	}
}
