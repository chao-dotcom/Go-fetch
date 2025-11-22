SHELL := /bin/sh
GO := go
API_ADDR ?= :8080
STORAGE_DRIVER ?= memory
BROKER_DRIVER ?= memory

.PHONY: run-api run-worker test docker-up docker-down lint

run-api:
	@echo "Starting API server..."
	@API_ADDR=$(API_ADDR) STORAGE_DRIVER=$(STORAGE_DRIVER) BROKER_DRIVER=$(BROKER_DRIVER) $(GO) run ./cmd/api

run-worker:
	@echo "Starting worker..."
	@STORAGE_DRIVER=$(STORAGE_DRIVER) BROKER_DRIVER=$(BROKER_DRIVER) $(GO) run ./cmd/worker

load-test:
	@echo "Running k6 load test (requires k6 CLI)..."
	k6 run scripts/load-test.js

test:
	@$(GO) test ./...

test-verbose:
	@$(GO) test -v ./...

test-race:
	@$(GO) test -race ./...

test-coverage:
	@$(GO) test -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-unit:
	@$(GO) test -v ./internal/worker/ ./internal/api/handlers/ ./internal/storage/ ./internal/broker/

test-bench:
	@$(GO) test -bench=. -benchmem ./...

quick-test:
	@echo "Running quick integration test..."
	@bash scripts/quick-test.sh

docker-up:
	@docker compose up --build

docker-down:
	@docker compose down -v

lint:
	@$(GO) vet ./...

proto:
	protoc --go_out=. --go-grpc_out=. internal/grpc/proto/taskqueue.proto

security-scan:
	@echo "Running gosec..."
	@command -v gosec >/dev/null 2>&1 || (echo "Installing gosec..." && GO111MODULE=on go install github.com/securego/gosec/v2/cmd/gosec@latest)
	@gosec ./...

