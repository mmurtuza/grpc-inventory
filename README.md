# gRPC Inventory Service

This project demonstrates a high-performance inventory microservice using gRPC in Go. It acts as an abstraction layer over an existing REST API to provide efficient, strictly-typed, and stream-capable endpoints for client applications.

## Project Structure

*   `proto/`: Contains the Protocol Buffer definition (`inventory.proto`) and the generated Go code (`proto/inventory`).
*   `server/`: The gRPC server implementation in Go. It handles Unary and Server-Streaming RPCs and proxies requests to the downstream REST API.
*   `client/`: A sample gRPC client in Go to test the server endpoints.
*   `rest-api/`: A mock REST API that serves as the backend data source for the gRPC server.

## Features

*   **Protocol Buffers (protobuf):** Defines a strict contract between the client and server.
*   **Unary RPC (`CheckStock`):** Fetches inventory details for a single SKU.
*   **Server-Streaming RPC (`StreamLowStock`):** Efficiently streams multiple low-stock items back to the client as they become available, reducing memory overhead.
*   **Middleware/Interceptors:** Includes unary interceptors for request logging and monitoring.
*   **Error Handling:** Translates HTTP status codes from the REST API into well-typed gRPC status codes (e.g., `NotFound`, `Internal`).
*   **Server Reflection:** Enabled for debugging and interacting with the server using tools like `grpcurl`.

## Prerequisites

*   [Go](https://go.dev/) (1.21+)
*   [Protocol Buffers Compiler (`protoc`)](https://grpc.io/docs/protoc-installation/)
*   gRPC Go plugins (`protoc-gen-go`, `protoc-gen-go-grpc`)

## Getting Started

1.  **Generate Protobuf Code**
    *(If you modify `inventory.proto`, you'll need to regenerate the Go code)*
    ```bash
    protoc --go_out=. --go_opt=paths=source_relative \
        --go-grpc_out=. --go-grpc_opt=paths=source_relative \
        proto/inventory.proto
    ```

2.  **Start the REST API**
    Navigate to the `rest-api` directory and start the server (typically runs on `http://localhost:8080`).

3.  **Start the gRPC Server**
    ```bash
    go run server/main.go
    ```
    The server will start listening on port `:50051`.

4.  **Run the Client**
    ```bash
    go run client/main.go
    ```

## Environment Variables

*   `REST_API_URL`: The base URL of the downstream REST API. Defaults to `http://localhost:8080`.
