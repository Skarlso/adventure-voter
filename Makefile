.PHONY: build run dev clean docker test help

# Variables
BINARY_NAME=server
DOCKER_IMAGE=kube-quest
GO_FILES=$(shell find . -name '*.go' -type f)

# Set the build dir, where built cross-compiled binaries will be output
BUILDDIR := bin

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Default target
help:
	@echo "🎮 Kubernetes Quest - Make targets:"
	@echo "  make build      - Build the server binary"
	@echo "  make run        - Build and run the server"
	@echo "  make dev        - Run with hot reload (requires air)"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make docker     - Build Docker image"
	@echo "  make test       - Run tests"
	@echo "  make deps       - Download dependencies"
	@echo ""
	@echo "Quick start: make run"

# Build the server
build:
	@echo "🔨 Building server..."
	@go build -o $(BINARY_NAME) ./backend/main.go
	@echo "✅ Build complete: ./$(BINARY_NAME)"

# Build and run
run: build
	@echo "🚀 Starting server..."
	@echo "🎬 Presenter: http://localhost:8080/presenter/"
	@echo "🎮 Voter: http://localhost:8080/voter/"
	@./$(BINARY_NAME)

# Development with hot reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "❌ air not found. Install with: go install github.com/cosmtrek/air@latest"; \
		exit 1; \
	fi

# Run tests
test:
	@echo "🧪 Running tests..."
	@go test -v ./...

# Download dependencies
deps:
	@echo "📦 Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✅ Dependencies ready"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf tmp/
	@echo "✅ Clean complete"

# Build Docker image
docker:
	@echo "🐳 Building Docker image..."
	@docker build -t $(DOCKER_IMAGE) .
	@echo "✅ Docker image built: $(DOCKER_IMAGE)"

# Run Docker container
docker-run:
	@echo "🐳 Running Docker container..."
	@docker run -p 8080:8080 $(DOCKER_IMAGE)

# Docker compose up
compose-up:
	@echo "🐳 Starting with docker-compose..."
	@docker-compose up --build

# Docker compose down
compose-down:
	@echo "🐳 Stopping docker-compose..."
	@docker-compose down

# Format code
fmt:
	@echo "🎨 Formatting code..."
	@go fmt ./...
	@echo "✅ Format complete"

# Lint code (requires golangci-lint)
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.6.0

golangci-lint: $(GOLANGCI_LINT)
$(GOLANGCI_LINT): $(LOCALBIN)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s $(GOLANGCI_LINT_VERSION)

lint: golangci-lint ## Run golangci-lint.
	$(GOLANGCI_LINT) run

# Build for production (with optimizations)
build-prod:
	@echo "🔨 Building for production..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY_NAME) backend/main.go
	@echo "✅ Production build complete"

# Create a new chapter template
new-chapter:
	@echo "Creating new chapter..."
	@echo "Use: content/chapters/04-my-chapter.md as template"

# Validate story files
validate:
	@echo "✅ Validation will happen when server starts"
	@./$(BINARY_NAME) -h > /dev/null 2>&1 || (echo "Build first with: make build" && exit 1)
