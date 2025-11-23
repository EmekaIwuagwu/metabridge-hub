.PHONY: help build test clean docker-build docker-up docker-down migrate deploy-contracts

# Default environment
ENV ?= testnet

# Colors for output
BLUE := \033[0;34m
GREEN := \033[0;32m
RED := \033[0;31m
NC := \033[0m # No Color

help: ## Show this help message
	@echo "$(BLUE)Articium Hub - Multi-Chain Bridge Protocol$(NC)"
	@echo ""
	@echo "$(GREEN)Available targets:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}'

build: ## Build all Go binaries
	@echo "$(GREEN)Building Go binaries...$(NC)"
	@mkdir -p bin
	CGO_ENABLED=0 go build -o bin/api ./cmd/api
	CGO_ENABLED=0 go build -o bin/relayer ./cmd/relayer
	CGO_ENABLED=0 go build -o bin/listener ./cmd/listener
	CGO_ENABLED=0 go build -o bin/batcher ./cmd/batcher
	CGO_ENABLED=0 go build -o bin/migrator ./cmd/migrator
	@echo "$(GREEN)Build complete! Binaries in ./bin/$(NC)"
	@ls -lh bin/

test: ## Run all tests
	@echo "$(GREEN)Running tests...$(NC)"
	go test -v ./...

test-integration: ## Run integration tests
	@echo "$(GREEN)Running integration tests...$(NC)"
	go test -v ./tests/integration/...

test-e2e: ## Run end-to-end tests
	@echo "$(GREEN)Running E2E tests...$(NC)"
	go test -v ./tests/e2e/...

clean: ## Clean build artifacts
	@echo "$(GREEN)Cleaning...$(NC)"
	rm -rf bin/
	rm -rf contracts/evm/node_modules
	rm -rf contracts/evm/artifacts
	rm -rf contracts/evm/cache

deps: ## Install dependencies
	@echo "$(GREEN)Installing Go dependencies...$(NC)"
	go mod download
	go mod verify

lint: ## Run linters
	@echo "$(GREEN)Running linters...$(NC)"
	golangci-lint run ./...

# Docker commands

docker-build: ## Build Docker images
	@echo "$(GREEN)Building Docker images...$(NC)"
	docker build -t articium-listener:latest --target listener .
	docker build -t articium-relayer:latest --target relayer .
	docker build -t articium-api:latest --target api .

docker-up: ## Start services with Docker Compose (testnet)
	@echo "$(GREEN)Starting services for $(ENV)...$(NC)"
	docker-compose -f docker-compose.$(ENV).yaml up -d

docker-down: ## Stop services with Docker Compose
	@echo "$(GREEN)Stopping services for $(ENV)...$(NC)"
	docker-compose -f docker-compose.$(ENV).yaml down

docker-logs: ## View Docker logs
	docker-compose -f docker-compose.$(ENV).yaml logs -f

docker-ps: ## Show running containers
	docker-compose -f docker-compose.$(ENV).yaml ps

# Database commands

migrate-up: ## Run database migrations
	@echo "$(GREEN)Running database migrations for $(ENV)...$(NC)"
	@if [ "$(ENV)" = "testnet" ]; then \
		psql -h localhost -U bridge_user -d articium_testnet -f internal/database/schema.sql; \
	elif [ "$(ENV)" = "mainnet" ]; then \
		psql -h $$DB_HOST -U bridge_user -d articium_mainnet -f internal/database/schema.sql; \
	else \
		psql -h localhost -U bridge_user -d articium_dev -f internal/database/schema.sql; \
	fi

migrate-down: ## Rollback database migrations
	@echo "$(RED)Rolling back migrations for $(ENV)...$(NC)"
	@echo "Not implemented - manual rollback required"

db-shell: ## Connect to database shell
	@if [ "$(ENV)" = "testnet" ]; then \
		psql -h localhost -U bridge_user -d articium_testnet; \
	elif [ "$(ENV)" = "mainnet" ]; then \
		psql -h $$DB_HOST -U bridge_user -d articium_mainnet; \
	else \
		psql -h localhost -U bridge_user -d articium_dev; \
	fi

# Smart contract commands

contracts-install: ## Install smart contract dependencies
	@echo "$(GREEN)Installing contract dependencies...$(NC)"
	cd contracts/evm && npm install

contracts-compile: ## Compile smart contracts
	@echo "$(GREEN)Compiling EVM contracts...$(NC)"
	cd contracts/evm && npx hardhat compile

contracts-test: ## Test smart contracts
	@echo "$(GREEN)Testing EVM contracts...$(NC)"
	cd contracts/evm && npx hardhat test

# Deployment commands

deploy-polygon-amoy: ## Deploy to Polygon Amoy testnet
	@echo "$(GREEN)Deploying to Polygon Amoy...$(NC)"
	cd contracts/evm && npx hardhat deploy --network polygon-amoy --tags Bridge

deploy-bnb-testnet: ## Deploy to BNB testnet
	@echo "$(GREEN)Deploying to BNB Testnet...$(NC)"
	cd contracts/evm && npx hardhat deploy --network bnb-testnet --tags Bridge

deploy-avalanche-fuji: ## Deploy to Avalanche Fuji
	@echo "$(GREEN)Deploying to Avalanche Fuji...$(NC)"
	cd contracts/evm && npx hardhat deploy --network avalanche-fuji --tags Bridge

deploy-ethereum-sepolia: ## Deploy to Ethereum Sepolia
	@echo "$(GREEN)Deploying to Ethereum Sepolia...$(NC)"
	cd contracts/evm && npx hardhat deploy --network ethereum-sepolia --tags Bridge

deploy-all-testnet: ## Deploy to all testnets
	@echo "$(GREEN)Deploying to all testnets...$(NC)"
	$(MAKE) deploy-polygon-amoy
	$(MAKE) deploy-bnb-testnet
	$(MAKE) deploy-avalanche-fuji
	$(MAKE) deploy-ethereum-sepolia

# Monitoring commands

prometheus: ## Open Prometheus dashboard
	@open http://localhost:9090 || xdg-open http://localhost:9090

grafana: ## Open Grafana dashboard
	@open http://localhost:3000 || xdg-open http://localhost:3000

# Development commands

dev: ## Start development environment
	@echo "$(GREEN)Starting development environment...$(NC)"
	$(MAKE) docker-up ENV=testnet
	@echo "$(GREEN)Services started!$(NC)"
	@echo "API: http://localhost:8080"
	@echo "Prometheus: http://localhost:9090"
	@echo "Grafana: http://localhost:3000"

stop: ## Stop development environment
	@echo "$(GREEN)Stopping development environment...$(NC)"
	$(MAKE) docker-down ENV=testnet

restart: ## Restart services
	$(MAKE) stop
	$(MAKE) dev

logs-listener: ## View listener logs
	docker logs -f articium-listener-$(ENV)

logs-relayer: ## View relayer logs
	docker logs -f articium-relayer-$(ENV)

logs-api: ## View API logs
	docker logs -f articium-api-$(ENV)

# Utility commands

fmt: ## Format Go code
	@echo "$(GREEN)Formatting code...$(NC)"
	go fmt ./...

vet: ## Run go vet
	@echo "$(GREEN)Running go vet...$(NC)"
	go vet ./...

mod-tidy: ## Tidy go modules
	@echo "$(GREEN)Tidying go modules...$(NC)"
	go mod tidy

check: fmt vet lint test ## Run all checks

version: ## Show version information
	@echo "Articium Hub v1.0.0"
	@echo "Go version: $$(go version)"
	@echo "Docker version: $$(docker --version)"

.DEFAULT_GOAL := help
