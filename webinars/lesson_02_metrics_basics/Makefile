GO = go
GOLANGCI_LINT = ~/go/bin/golangci-lint
PACKAGES = ./...

.PHONY: all lint fmt vet test tidy run

all: lint test fmt vet tidy run

lint: ## Run code linters
	@echo "Running golangci-lint..."
	@$(GOLANGCI_LINT) run

fmt: ## Check and fix formatting
	@echo "Checking formatting..."
	@$(GO) fmt $(PACKAGES)

vet: ## Run go vet for static analysis
	@echo "Running go vet..."
	@$(GO) vet $(PACKAGES)

test: ## Run unit tests
	@echo "Running tests..."
	@$(GO) test -v $(PACKAGES)

tidy: ## Run go mod tidy to clean up dependencies
	@echo "Tidying up modules..."
	@$(GO) mod tidy

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

run: ## Start application
	@echo "Starting application"
	docker compose up -d --build --force-recreate

stop: ## Stop application
	@echo "Stopping application"
	docker compose down -v