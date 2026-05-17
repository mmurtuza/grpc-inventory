# syntax=docker/dockerfile:1

# ─── Build Stage ─────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build all three binaries as fully static executables.
# CGO_ENABLED=0 + -ldflags="-s -w" produces small, portable binaries.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
      -ldflags="-s -w" -o /bin/rest-api ./cmd/rest-api

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
      -ldflags="-s -w" -o /bin/server ./cmd/server

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
      -ldflags="-s -w" -o /bin/client ./cmd/client

# ─── REST API Container ───────────────────────────────────────────────────────
# Uses alpine so Docker health checks (wget /healthz) work.
FROM alpine:3.21 AS rest-api
RUN addgroup -S app && adduser -S app -G app
COPY --from=builder /bin/rest-api /rest-api
EXPOSE 8080
USER app
ENTRYPOINT ["/rest-api"]

# ─── gRPC Server Container ────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot AS server
COPY --from=builder /bin/server /server
EXPOSE 50051
USER nonroot:nonroot
ENTRYPOINT ["/server"]

# ─── gRPC Client Container ───────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot AS client
COPY --from=builder /bin/client /client
USER nonroot:nonroot
ENTRYPOINT ["/client"]
