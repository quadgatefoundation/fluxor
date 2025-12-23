.PHONY: help build run test clean docker-up docker-down migrate

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "Building Todo API..."
	go build -o todo-api ./main.go

run: ## Run the application
	@echo "Running Todo API..."
	@./run.sh

docker-up: ## Start PostgreSQL with Docker Compose
	@echo "Starting PostgreSQL..."
	docker-compose up -d
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5
	@until docker exec todo-api-postgres pg_isready -U todo_user > /dev/null 2>&1; do \
		echo "   Waiting for PostgreSQL..."; \
		sleep 1; \
	done
	@echo "✅ PostgreSQL is ready!"

docker-down: ## Stop PostgreSQL
	@echo "Stopping PostgreSQL..."
	docker-compose down

migrate: ## Run database migrations
	@echo "Running migrations..."
	@docker exec -i todo-api-postgres psql -U todo_user -d todo_db < migrations/001_init.sql

test: ## Run tests
	@echo "Running tests..."
	go test ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f todo-api
	go clean
