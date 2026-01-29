.PHONY: help build run test clean docker-up docker-down

help: ## Show this help message
	@echo "Available commands:"
	@echo "  make build       - Build the application"
	@echo "  make run         - Run the application"
	@echo "  make test        - Run tests"
	@echo "  make docker-up   - Start Docker containers"
	@echo "  make docker-down - Stop Docker containers"

build: ## Build the application
	go build -o bin/threatlog.exe cmd/server/main.go

run: ## Run the application
	go run cmd/server/main.go

test: ## Run tests
	go test -v -race -cover ./...

clean: ## Clean build artifacts
	rm -rf bin/

docker-up: ## Start Docker containers
	docker-compose -f docker/docker-compose.yml up -d

docker-down: ## Stop Docker containers
	docker-compose -f docker/docker-compose.yml down