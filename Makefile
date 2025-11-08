.PHONY: build test lint clean run fmt vet race coverage install help

# Build variables
BINARY_NAME=aistack
BUILD_DIR=./dist
CMD_DIR=./cmd/aistack
GO_FILES=$(shell find . -name '*.go' -not -path './vendor/*')

# Build flags
GO_BUILD_FLAGS=-tags netgo
CGO_ENABLED=0
LDFLAGS=-ldflags "-s -w"

# CUDA build flags (for GPU support)
CUDA_BUILD_FLAGS=-tags "netgo,cuda"
CUDA_CGO_ENABLED=1

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary (no GPU support)
	@echo "Building $(BINARY_NAME) (no GPU support)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

build-cuda: ## Build the binary with CUDA/GPU support (requires NVIDIA CUDA Toolkit)
	@echo "Building $(BINARY_NAME) with CUDA support..."
	@echo "Checking for NVIDIA GPU..."
	@if ! command -v nvidia-smi >/dev/null 2>&1; then \
		echo "ERROR: nvidia-smi not found. Install NVIDIA drivers first."; \
		exit 1; \
	fi
	@echo "Checking for CUDA Toolkit..."
	@if [ ! -d "/usr/local/cuda" ] && [ ! -d "/usr/lib/cuda" ]; then \
		echo "ERROR: CUDA Toolkit not found at /usr/local/cuda or /usr/lib/cuda"; \
		echo "Install: sudo apt install nvidia-cuda-toolkit"; \
		exit 1; \
	fi
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CUDA_CGO_ENABLED) go build $(CUDA_BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Binary built with CUDA support: $(BUILD_DIR)/$(BINARY_NAME)"

test: ## Run unit tests
	@echo "Running tests..."
	go test ./... -v

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
