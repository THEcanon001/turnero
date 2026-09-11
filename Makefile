.PHONY: build run test test-cover lint migrate-up migrate-down docker-up docker-down clean

APP_NAME := turnero
MAIN_PATH := ./cmd/server

# Build
build:
	go build -o bin/$(APP_NAME) $(MAIN_PATH)

# Run
run:
	go run $(MAIN_PATH)

# Test
test:
	go test ./... -v -race -count=1

test-cover:
	go test ./... -v -race -count=1 -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Lint
lint:
	go vet ./...
	staticcheck ./...

# Docker
docker-up:
	docker compose up -d

docker-down:
	docker compose down

# Migrations (requires running database)
migrate-up:
	go run $(MAIN_PATH) -migrate-only

migrate-down:
	@echo "Use golang-migrate CLI: migrate -path internal/platform/database/migrations -database 'postgres://turnero:turnero_dev@localhost:5432/turnero?sslmode=disable' down 1"

# Clean
clean:
	rm -rf bin/ coverage.out coverage.html
