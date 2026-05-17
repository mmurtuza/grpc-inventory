.PHONY: proto rest server client build test test-coverage lint fmt tidy sqlc db-seed db-reset \
        docker-up docker-down docker-build all stop-all

# ─── Proto ───────────────────────────────────────────────────────────────────
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/inventory.proto

# ─── sqlc ─────────────────────────────────────────────────────────────────────
sqlc:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate

# ─── Database helpers ─────────────────────────────────────────────────────────
db-seed:
	psql "$(DATABASE_URL)" -f db/seed.sql

db-reset:
	psql "$(DATABASE_URL)" -c "DROP TABLE IF EXISTS inventory;"
	psql "$(DATABASE_URL)" -f db/schema.sql
	psql "$(DATABASE_URL)" -f db/seed.sql

# ─── Local Development ────────────────────────────────────────────────────────
rest:
	go run ./cmd/rest-api

server:
	go run ./cmd/server

client:
	go run ./cmd/client

all:
	@echo "Starting REST API and gRPC server..."
	@go run ./cmd/rest-api & REST_PID=$$!; \
	sleep 1; \
	go run ./cmd/server & SERVER_PID=$$!; \
	sleep 1; \
	go run ./cmd/client; \
	kill $$REST_PID $$SERVER_PID 2>/dev/null; \
	echo "Done."

stop-all:
	@pkill -f "cmd/rest-api" || true
	@pkill -f "cmd/server" || true
	@pkill -f "cmd/client" || true

# ─── Build ────────────────────────────────────────────────────────────────────
build:
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/rest-api ./cmd/rest-api
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/server   ./cmd/server
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/client   ./cmd/client
	@echo "Binaries in ./bin/"

# ─── Testing ──────────────────────────────────────────────────────────────────
test:
	go test -race ./...

test-coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Report: coverage.html"

# ─── Code Quality ─────────────────────────────────────────────────────────────
fmt:
	go fmt ./...

tidy:
	go mod tidy

lint:
	go vet ./...
	@command -v staticcheck >/dev/null 2>&1 && staticcheck ./... || \
		echo "staticcheck not installed: go install honnef.co/go/tools/cmd/staticcheck@latest"

# ─── Docker ───────────────────────────────────────────────────────────────────
docker-build:
	docker compose build

docker-up:
	docker compose up --build

docker-down:
	docker compose down -v
