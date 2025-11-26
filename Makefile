.PHONY: build test lint clean run fmt vet race coverage install help

# Build variables
BINARY_NAME=aistack
BUILD_DIR=./dist
CMD_DIR=./cmd/aistack
GO_FILES=$(shell find . -name '*.go' -not -path './vendor/*')

# Build flags
LDFLAGS=-ldflags "-s -w"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary (auto-detects CUDA if available)
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@if command -v nvidia-smi >/dev/null 2>&1 && ([ -d "/usr/local/cuda" ] || [ -d "/usr/lib/cuda" ]); then \
		echo "CUDA detected - building with GPU support"; \
		CGO_ENABLED=1 go build -tags "netgo,cuda" $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR); \
	else \
		echo "No CUDA detected - building without GPU support"; \
		CGO_ENABLED=0 go build -tags netgo $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR); \
	fi
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

build-no-cuda: ## Build the binary without CUDA support (force)
	@echo "Building $(BINARY_NAME) without CUDA support (forced)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -tags netgo $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

test: ## Run unit tests
	@echo "Running tests..."
	@if command -v nvidia-smi >/dev/null 2>&1 && ([ -d "/usr/local/cuda" ] || [ -d "/usr/lib/cuda" ]); then \
		echo "CUDA detected - running tests with GPU support"; \
		CGO_ENABLED=1 go test -tags cuda ./... -v; \
	else \
		echo "No CUDA detected - running tests without GPU support"; \
		go test ./... -v; \
	fi

race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	go test ./... -race

coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test ./... -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

lint: fmt vet ## Run linters (fmt + vet)
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping..."; \
		echo "Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

run: ## Run the application
	@echo "Running $(BINARY_NAME)..."
	go run $(CMD_DIR)

install: build ## Install the binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@go clean

tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	go mod tidy

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download

all: clean deps lint test build ## Run all checks and build
