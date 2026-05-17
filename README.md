# gRPC Inventory Service

![CI](https://github.com/murtuza/grpc-inventory/actions/workflows/ci.yml/badge.svg)

A high-performance pharmacy inventory microservice built with **gRPC** and **Go**. It acts as an abstraction layer over an existing REST API, exposing strictly-typed, stream-capable endpoints to client applications.

## Project Structure

```
.
├── proto/                   # Protobuf definition + generated Go code
│   ├── inventory.proto
│   └── inventory/           # Generated: inventory.pb.go, inventory_grpc.pb.go
├── rest-api/                # Mock REST API (in-memory data store)
│   ├── main.go
│   └── main_test.go
├── server/                  # gRPC server (proxies to REST API)
│   ├── main.go
│   └── main_test.go
├── client/                  # Sample gRPC client (demo / smoke test)
│   └── main.go
├── .github/workflows/       # GitHub Actions CI pipeline
├── Dockerfile               # Multi-stage build (distroless runtime)
├── docker-compose.yml       # Orchestrates all three services
├── Makefile                 # Developer task runner
├── .env.example             # Environment variable reference
└── postman_collection.json  # Postman collection for manual testing
```

## Features

- **Protocol Buffers (protobuf):** Strict, versioned contract between client and server.
- **Unary RPC (`CheckStock`):** Fetch inventory details for a single SKU.
- **Server-Streaming RPC (`StreamLowStock`):** Stream all low-stock items to the client.
- **Middleware / Interceptors:** Unary logging, panic recovery, and server-streaming logging interceptors.
- **Input Validation:** SKU presence and non-negative quantity enforced on all write operations.
- **Graceful Shutdown:** SIGTERM/SIGINT handlers drain in-flight requests before exiting.
- **Structured Logging:** `log/slog` with JSON output in production (`LOG_FORMAT=json`).
- **Health Check:** `GET /healthz` on the REST API for Docker and load-balancer probes.
- **CORS Support:** Configurable via `ALLOWED_ORIGIN` environment variable.
- **Keepalive:** Client and server keepalive parameters prevent zombie connections.
- **Server Reflection:** Enabled for `grpcurl` and gRPC UI debugging.
- **Distroless Docker Images:** Minimal attack surface, non-root runtime user.

## Prerequisites

| Tool | Version | Notes |
|------|---------|-------|
| [Go](https://go.dev/) | 1.22+ | Required for `http.ServeMux` pattern matching |
| [protoc](https://grpc.io/docs/protoc-installation/) | Latest | Only needed to regenerate proto code |
| `protoc-gen-go` | Latest | `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest` |
| `protoc-gen-go-grpc` | Latest | `go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest` |
| [Docker](https://docs.docker.com/get-docker/) | 20.10+ | For containerised runs |

## Getting Started

### Local Development (without Docker)

```bash
# 1. Install dependencies
go mod download

# 2. Start the REST API (terminal 1)
make rest

# 3. Start the gRPC server (terminal 2)
make server

# 4. Run the demo client (terminal 3)
make client

# Or run everything in one command:
make all
```

### Docker Compose

```bash
# Copy and customise environment variables (optional)
cp .env.example .env

# Build and start all three services
make docker-up

# Tear down
make docker-down
```

The client container will connect to the gRPC server, run both demo RPCs, print results, and exit.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `REST_API_PORT` | `8080` | Port the REST API listens on |
| `LOG_FORMAT` | `text` | Log format: `text` (dev) or `json` (prod) |
| `ALLOWED_ORIGIN` | `*` | CORS allowed origin |
| `REST_API_URL` | `http://localhost:8080` | REST API base URL used by the gRPC server |
| `GRPC_PORT` | `50051` | Port the gRPC server listens on |
| `SERVER_ADDR` | `localhost:50051` | gRPC server address used by the client |

See [`.env.example`](.env.example) for a full annotated reference.

## Testing

```bash
# Run all tests with the race detector
make test

# Generate an HTML coverage report (opens coverage.html)
make test-coverage

# Static analysis
make lint
```

## API Reference

### REST API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/healthz` | Health check |
| `GET` | `/api/inventory/{sku}` | Get a single stock item |
| `GET` | `/api/inventory/low-stock` | List items below stock threshold (50) |
| `POST` | `/api/inventory` | Add a new stock item |
| `PUT` | `/api/inventory/{sku}` | Update an existing stock item |
| `DELETE` | `/api/inventory/{sku}` | Remove a stock item |

### gRPC Service (`inventory.InventoryService`)

| RPC | Type | Description |
|-----|------|-------------|
| `CheckStock` | Unary | Fetch stock for a single SKU |
| `StreamLowStock` | Server-Streaming | Stream all low-stock items |

Use [`grpcurl`](https://github.com/fullstorydev/grpcurl) to call the server directly:

```bash
# Unary call
grpcurl -plaintext -d '{"sku":"MED-001"}' localhost:50051 inventory.InventoryService/CheckStock

# Streaming call
grpcurl -plaintext localhost:50051 inventory.InventoryService/StreamLowStock
```

## Build

```bash
# Compile all binaries into ./bin/
make build
```

## Regenerating Protobuf Code

Only required if you modify `proto/inventory.proto`:

```bash
make proto
```
