# Makefile for Movie Rating API

.PHONY: all build docker-up docker-down docker-build test-e2e clean proto init

# Variables
DOCKER_COMPOSE=docker compose
APP_NAME=robin-camp
PROTO_DIR=./src/api

# Default target
all: proto build

# Initialize project dependencies
init:
	@echo "Installing dependencies..."
	cd src && go mod tidy
	cd src && go mod download

# Generate protobuf files
proto:
	@echo "Generating protobuf files..."
	cd src && make api

# Build the application locally
build:
	@echo "Building application..."
	cd src && go build -o ../bin/server ./cmd/src

# Build and start all containers
docker-up:
	@echo "Starting containers..."
	$(DOCKER_COMPOSE) up -d --build
	@echo "Waiting for services to be healthy..."
	@sleep 5
	@docker compose ps

# Stop and remove containers
docker-down:
	@echo "Stopping containers..."
	$(DOCKER_COMPOSE) down -v

# Build docker images
docker-build:
	@echo "Building docker images..."
	$(DOCKER_COMPOSE) build

# Run end-to-end tests
test-e2e:
	@echo "Running E2E tests..."
	@if [ ! -f .env ]; then \
		echo "Error: .env file not found. Please copy .env.example to .env and fill in the values."; \
		exit 1; \
	fi
	@chmod +x ./e2e-test.sh
	@./e2e-test.sh

# Clean up build artifacts
clean:
	@echo "Cleaning up..."
	rm -rf bin/
	$(DOCKER_COMPOSE) down -v
	docker system prune -f

# Run tests
test:
	cd src && go test -v ./...

# Run with hot reload (requires air)
dev:
	cd src && air

# Check service health
health:
	@curl -s http://localhost:8080/healthz || echo "Service not healthy"
