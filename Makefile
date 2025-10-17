# Makefile for Movie Rating API

.PHONY: docker-up docker-down test-e2e

# Build and start all containers
docker-up:
	@echo "Starting containers..."
	docker compose up -d --build
	@echo "Waiting for services to be healthy..."
	@sleep 5
	@docker compose ps

# Stop and remove containers
docker-down:
	@echo "Stopping containers..."
	docker compose down -v

# Run end-to-end tests
test-e2e:
	@echo "Running E2E tests..."
	@if [ ! -f .env ]; then \
		echo "Error: .env file not found. Please copy .env.example to .env and fill in the values."; \
		exit 1; \
	fi
	@chmod +x ./e2e-test.sh
	@./e2e-test.sh
