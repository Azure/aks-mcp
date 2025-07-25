# AKS-MCP Makefile
# Provides convenience targets for development, building, testing, and releasing

# Variables
BINARY_NAME = aks-mcp
MAIN_PATH = ./cmd/aks-mcp
DOCKER_IMAGE = aks-mcp
DOCKER_TAG ?= latest

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_TREE_STATE ?= $(shell if git diff --quiet 2>/dev/null; then echo "clean"; else echo "dirty"; fi)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go build flags
LDFLAGS = -ldflags "-X github.com/Azure/aks-mcp/internal/version.GitVersion=$(VERSION) \
                   -X github.com/Azure/aks-mcp/internal/version.GitCommit=$(GIT_COMMIT) \
                   -X github.com/Azure/aks-mcp/internal/version.GitTreeState=$(GIT_TREE_STATE) \
                   -X github.com/Azure/aks-mcp/internal/version.BuildMetadata=$(BUILD_DATE)"

# Build options
BUILD_FLAGS = -trimpath -tags withoutebpf
CGO_ENABLED ?= 0

# Platform targets for cross-compilation
PLATFORMS = \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64 \
	windows/arm64

# Default target
.DEFAULT_GOAL := help

##@ Development

.PHONY: deps
deps: ## Download and verify dependencies
	@echo "==> Downloading dependencies..."
	go mod download
	go mod verify

.PHONY: build
build: deps ## Build the binary
	@echo "==> Building $(BINARY_NAME)..."
	CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)

.PHONY: install
install: ## Install the binary to GOBIN
	@echo "==> Installing $(BINARY_NAME)..."
	CGO_ENABLED=$(CGO_ENABLED) go install $(BUILD_FLAGS) $(LDFLAGS) $(MAIN_PATH)

.PHONY: clean
clean: ## Clean build artifacts
	@echo "==> Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	rm -rf dist/
	rm -f coverage.txt
	go clean -cache
	docker image rm -f $(DOCKER_IMAGE):$(DOCKER_TAG) 2>/dev/null || true

##@ Testing

.PHONY: test
test: ## Run tests
	@echo "==> Running tests..."
	go test ./...

.PHONY: test-race
test-race: ## Run tests with race detection
	@echo "==> Running tests with race detection..."
	go test -race ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "==> Running tests with coverage..."
	go test -race -coverprofile=coverage.txt -covermode=atomic ./...

.PHONY: test-verbose
test-verbose: ## Run tests in verbose mode
	@echo "==> Running tests (verbose)..."
	go test -v ./...

##@ Code Quality

.PHONY: fmt
fmt: ## Format code
	@echo "==> Formatting code..."
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	@echo "==> Running go vet..."
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	@echo "==> Running golangci-lint..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@$$(go env GOPATH)/bin/golangci-lint run --timeout=5m

.PHONY: lint-fix
lint-fix: ## Run golangci-lint with auto-fix
	@echo "==> Running golangci-lint with auto-fix..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@$$(go env GOPATH)/bin/golangci-lint run --fix --timeout=5m

##@ Docker

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "==> Building Docker image..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_TREE_STATE=$(GIT_TREE_STATE) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) .

.PHONY: docker-run
docker-run: docker-build ## Run Docker container
	@echo "==> Running Docker container..."
	docker run --rm $(DOCKER_IMAGE):$(DOCKER_TAG) --help

.PHONY: docker-shell
docker-shell: docker-build ## Run Docker container with shell access
	@echo "==> Running Docker container with shell..."
	docker run --rm -it --entrypoint=/bin/bash $(DOCKER_IMAGE):$(DOCKER_TAG)

##@ Release

.PHONY: release
release: clean ## Build for all platforms
	@echo "==> Building for all platforms..."
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d'/' -f1); \
		GOARCH=$$(echo $$platform | cut -d'/' -f2); \
		output_name=$(BINARY_NAME)-$$GOOS-$$GOARCH; \
		if [ "$$GOOS" = "windows" ]; then \
			output_name="$$output_name.exe"; \
		fi; \
		echo "Building for $$GOOS/$$GOARCH..."; \
		CGO_ENABLED=$(CGO_ENABLED) GOOS=$$GOOS GOARCH=$$GOARCH \
			go build $(BUILD_FLAGS) $(LDFLAGS) -o dist/$$output_name $(MAIN_PATH); \
	done

.PHONY: checksums
checksums: release ## Generate checksums for release binaries
	@echo "==> Generating checksums..."
	@cd dist && sha256sum * > checksums.txt

##@ Verification

.PHONY: check
check: fmt vet lint test ## Run all checks (format, vet, lint, test)

.PHONY: ci
ci: check test-coverage ## Run CI pipeline locally

##@ Utilities

.PHONY: version
version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Git Tree State: $(GIT_TREE_STATE)"
	@echo "Build Date: $(BUILD_DATE)"

.PHONY: info
info: ## Show build information
	@echo "Binary Name: $(BINARY_NAME)"
	@echo "Main Path: $(MAIN_PATH)"
	@echo "Docker Image: $(DOCKER_IMAGE):$(DOCKER_TAG)"
	@echo "CGO Enabled: $(CGO_ENABLED)"
	@echo "LDFLAGS: $(LDFLAGS)"

.PHONY: run
run: build ## Build and run the application with --help
	@echo "==> Running $(BINARY_NAME)..."
	./$(BINARY_NAME) --help

##@ Help

.PHONY: help
help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
