.PHONY: help build run test clean docker-up docker-down seed load-test

help: ## Show this help message
	@echo "Available commands:"
	@echo "  make build       - Build the application"
	@echo "  make run         - Run the application"
	@echo "  make test        - Run tests"
	@echo "  make seed        - Generate sample data"
	@echo "  make load-test   - Run load test"
	@echo "  make docker-up   - Start Docker containers"
	@echo "  make docker-down - Stop Docker containers"
	@echo "  make clean       - Clean build artifacts"

build: ## Build the application
	go build -o bin/threatlog.exe cmd/server/main.go

run: build ## Run the application
	.\bin\threatlog.exe

test: ## Run tests
	go test -v -race -cover ./...

seed: ## Generate sample data (10,000 logs)
	go run scripts/seed_data.go -count 10000

load-test: ## Run load test
	go run scripts/loadtest.go -concurrency 100 -duration 60 -rps 1000

docker-up: ## Start Docker containers
	cd docker && docker-compose up -d

docker-down: ## Stop Docker containers
	cd docker && docker-compose down

clean: ## Clean build artifacts
	rm -rf bin/