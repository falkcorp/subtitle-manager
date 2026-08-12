	# file: Makefile
# Makefile for Subtitle Manager - Comprehensive build automation

	# Project Configuration
	APP_NAME := subtitle-manager
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
GO_VERSION := $(shell go version | awk '{print $$3}')

# Build Configuration
BINARY_NAME := $(APP_NAME)
BINARY_PATH := ./bin/$(BINARY_NAME)
DOCKER_IMAGE := ghcr.io/jdfalk/$(APP_NAME)
DOCKER_TAG := $(VERSION)

PLATFORMS := linux/amd64,linux/arm64
# Go Build Flags
LDFLAGS := -ldflags="-s -w -X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)' -X 'main.GitCommit=$(GIT_COMMIT)'"
GO_BUILD_FLAGS := -v $(LDFLAGS)
CGO_ENABLED := 0

# Directories
WEBUI_DIR := ./webui
DIST_DIR := $(WEBUI_DIR)/dist
BIN_DIR := ./bin
DOCS_DIR := ./docs
PROTO_DIR := ./proto

# Tools
GOFMT := gofmt
GOLINT := golangci-lint
PROTOC := protoc

# Colors for output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m
COLOR_CYAN := \033[36m

# Default target
.PHONY: help
help: ## Show this help message
	@echo "$(COLOR_BOLD)$(APP_NAME) Build System$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_CYAN)Available targets:$(COLOR_RESET)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(COLOR_GREEN)%-20s$(COLOR_RESET) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(COLOR_YELLOW)Examples:$(COLOR_RESET)"
	@echo "  make build              # Build the application"
	@echo "  make dev-air            # Start development with live reloading (full build)"
	@echo "  make dev-air-fast       # Start development quickly (assumes web UI built)"
	@echo "  make webui-rebuild      # Force rebuild web UI when JS changes don't trigger"
	@echo "  make docker             # Build Docker image (includes protobuf generation)"
	@echo "  make proto-gen          # Generate protobuf code only"
	@echo "  make test-all           # Run all tests"
	@echo "  make test-race          # Run all tests with race detection"
	@echo "  make clean-all          # Clean everything"

#
# Build Targets
#

.PHONY: build
build: webui-build go-generate binary ## Build the complete application

.PHONY: binary
binary: $(BINARY_PATH) ## Build the Go binary

$(BINARY_PATH): $(shell find . -name '*.go' -not -path './webui/node_modules/*') go.mod go.sum
	@echo "$(COLOR_BLUE)Building Go binary...$(COLOR_RESET)"
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) -o $@ .
	@echo "$(COLOR_GREEN)✓ Binary built: $@$(COLOR_RESET)"

.PHONY: build-race
build-race: ## Build binary with race detection
	@echo "$(COLOR_BLUE)Building with race detection...$(COLOR_RESET)"
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=1 go build -race $(GO_BUILD_FLAGS) -o $(BINARY_PATH)-race .

.PHONY: build-static
build-static: ## Build static binary (no CGO)
	@echo "$(COLOR_BLUE)Building static binary...$(COLOR_RESET)"
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o $(BINARY_PATH)-static .

# build-sqlite was removed: SQLite is now compiled into every build via the
# pure-Go modernc.org/sqlite driver, so `make build` produces a binary that can
# serve the web UI on any platform. Use `make build`.

.PHONY: build-all-platforms
build-all-platforms: ## Build for multiple platforms
	@echo "$(COLOR_BLUE)Building for multiple platforms...$(COLOR_RESET)"
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 .
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@echo "$(COLOR_GREEN)✓ Multi-platform builds complete$(COLOR_RESET)"

#
# Web UI Targets
#

.PHONY: webui-deps
webui-deps: $(WEBUI_DIR)/node_modules ## Install web UI dependencies

$(WEBUI_DIR)/node_modules: $(WEBUI_DIR)/package.json $(WEBUI_DIR)/package-lock.json
	@echo "$(COLOR_BLUE)Installing web UI dependencies...$(COLOR_RESET)"
	cd $(WEBUI_DIR) && npm ci --legacy-peer-deps
	@touch $@

.PHONY: webui-deps-clean
webui-deps-clean: ## Clean install web UI dependencies (fixes version conflicts)
	@echo "$(COLOR_BLUE)Clean installing web UI dependencies...$(COLOR_RESET)"
	rm -rf $(WEBUI_DIR)/node_modules $(WEBUI_DIR)/package-lock.json
	cd $(WEBUI_DIR) && npm install --legacy-peer-deps
	@echo "$(COLOR_GREEN)✓ Web UI dependencies clean installed$(COLOR_RESET)"

.PHONY: webui-build
webui-build: webui-deps ## Build the web UI (always rebuilds to ensure fresh assets)
	@echo "$(COLOR_BLUE)Building web UI...$(COLOR_RESET)"
	cd $(WEBUI_DIR) && npm run build
	@echo "$(COLOR_GREEN)✓ Web UI built$(COLOR_RESET)"

# Create dist directory if it doesn't exist (for initial setup)
$(DIST_DIR):
	@mkdir -p $(DIST_DIR)

.PHONY: webui-dev
webui-dev: webui-deps ## Start web UI development server
	@echo "$(COLOR_BLUE)Starting web UI development server...$(COLOR_RESET)"
	cd $(WEBUI_DIR) && npm run dev

.PHONY: webui-lint
webui-lint: webui-deps ## Lint web UI code
	@echo "$(COLOR_BLUE)Linting web UI...$(COLOR_RESET)"
	cd $(WEBUI_DIR) && npm run lint

.PHONY: webui-test
webui-test: webui-deps ## Run web UI unit tests
	@echo "$(COLOR_BLUE)Running web UI tests...$(COLOR_RESET)"
	cd $(WEBUI_DIR) && npm test

.PHONY: webui-test-e2e
webui-test-e2e: webui-deps ## Run web UI end-to-end tests
	@echo "$(COLOR_BLUE)Running web UI E2E tests...$(COLOR_RESET)"
	cd $(WEBUI_DIR) && npm run test:e2e

#
# Go Code Management
#

.PHONY: go-generate
go-generate: webui-build ## Run go generate to embed web assets
	@echo "$(COLOR_BLUE)Running go generate...$(COLOR_RESET)"
	go generate ./webui
	@echo "$(COLOR_GREEN)✓ Go generate complete$(COLOR_RESET)"

.PHONY: go-mod-tidy
go-mod-tidy: ## Tidy Go modules
	@echo "$(COLOR_BLUE)Tidying Go modules...$(COLOR_RESET)"
	go mod tidy
	@echo "$(COLOR_GREEN)✓ Go modules tidied$(COLOR_RESET)"

.PHONY: go-mod-download
go-mod-download: ## Download Go modules
	@echo "$(COLOR_BLUE)Downloading Go modules...$(COLOR_RESET)"
	go mod download
	@echo "$(COLOR_GREEN)✓ Go modules downloaded$(COLOR_RESET)"

.PHONY: go-fmt
go-fmt: ## Format Go code
	@echo "$(COLOR_BLUE)Formatting Go code...$(COLOR_RESET)"
	$(GOFMT) -s -w .
	@echo "$(COLOR_GREEN)✓ Go code formatted$(COLOR_RESET)"

.PHONY: go-vet
go-vet: ## Run go vet
	@echo "$(COLOR_BLUE)Running go vet...$(COLOR_RESET)"
	go vet ./...
	@echo "$(COLOR_GREEN)✓ Go vet passed$(COLOR_RESET)"

.PHONY: go-lint
go-lint: ## Run golangci-lint
	@echo "$(COLOR_BLUE)Running golangci-lint...$(COLOR_RESET)"
	$(GOLINT) run ./...
	@echo "$(COLOR_GREEN)✓ Go lint passed$(COLOR_RESET)"

#
# Testing Targets
#

.PHONY: test
test: ## Run Go tests
	@echo "$(COLOR_BLUE)Running Go tests...$(COLOR_RESET)"
	GOTOOLCHAIN=local go test -v ./...

.PHONY: test-race
test-race: ## Run Go tests with race detection
	@echo "$(COLOR_BLUE)Running Go tests with race detection...$(COLOR_RESET)"
	GOTOOLCHAIN=local go test -race -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "$(COLOR_BLUE)Running tests with coverage...$(COLOR_RESET)"
	GOTOOLCHAIN=local go test -coverprofile=coverage.out -v ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(COLOR_GREEN)✓ Coverage report generated: coverage.html$(COLOR_RESET)"

.PHONY: test-all
test-all: test webui-test ## Run all tests (Go + Web UI)
	@echo "$(COLOR_GREEN)✓ All tests completed$(COLOR_RESET)"

.PHONY: test-e2e-all
test-e2e-all: test-race webui-test-e2e ## Run all tests including E2E
	@echo "$(COLOR_GREEN)✓ All tests including E2E completed$(COLOR_RESET)"

#
# Docker Targets
#

.PHONY: docker
docker: proto-gen ## Build Docker image (with protobuf generation)
	@echo "$(COLOR_BLUE)Building Docker image...$(COLOR_RESET)"
	docker build --pull \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) . || { \
			echo "$(COLOR_YELLOW)⚠️  WARNING: Docker build cache unavailable (GitHub Actions cache error). Continuing without cache.$(COLOR_RESET)"; \
		}
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest
	@echo "$(COLOR_GREEN)✓ Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)$(COLOR_RESET)"

.PHONY: docker-build-args
docker-build-args: proto-gen ## Build Docker image with build arguments (with protobuf generation)
	@echo "$(COLOR_BLUE)Building Docker image with build args...$(COLOR_RESET)"
	docker build --pull \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) . || { \
			echo "$(COLOR_YELLOW)⚠️  WARNING: Docker build cache unavailable (GitHub Actions cache error). Continuing without cache.$(COLOR_RESET)"; \
		}

.PHONY: docker-run
docker-run: docker ## Build and run Docker container
	@echo "$(COLOR_BLUE)Stopping and removing any existing subtitle-manager containers...$(COLOR_RESET)"
	@docker ps -q --filter "ancestor=$(DOCKER_IMAGE):$(DOCKER_TAG)" | xargs -r docker stop || true
	@docker ps -aq --filter "ancestor=$(DOCKER_IMAGE):$(DOCKER_TAG)" | xargs -r docker rm || true
	@docker ps -q --filter "name=$(APP_NAME)" | xargs -r docker stop || true
	@docker ps -aq --filter "name=$(APP_NAME)" | xargs -r docker rm || true
	@echo "$(COLOR_BLUE)Running Docker container in detached mode...$(COLOR_RESET)"
	docker run -d --rm --name $(APP_NAME) -p 8080:8080 -v $(APP_NAME):/config $(DOCKER_IMAGE):$(DOCKER_TAG)
	@echo "$(COLOR_GREEN)✓ Container started successfully$(COLOR_RESET)"
	@echo "$(COLOR_CYAN)Application available at: http://localhost:8080$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Container commands:$(COLOR_RESET)"
	@echo "  View logs:    docker logs $(APP_NAME)"
	@echo "  Follow logs:  docker logs -f $(APP_NAME)"
	@echo "  Stop:         docker stop $(APP_NAME)"
	@echo "  Shell access: docker exec -it $(APP_NAME) /bin/sh"
	@echo "$(COLOR_YELLOW)Volume mounted: $(APP_NAME) -> /app/data$(COLOR_RESET)"

.PHONY: docker-run-clean
docker-run-clean: ## Clean recreate volume and run Docker container
	@echo "$(COLOR_BLUE)Stopping and removing any existing subtitle-manager containers...$(COLOR_RESET)"
	@docker ps -q --filter "ancestor=$(DOCKER_IMAGE):$(DOCKER_TAG)" | xargs -r docker stop || true
	@docker ps -aq --filter "ancestor=$(DOCKER_IMAGE):$(DOCKER_TAG)" | xargs -r docker rm || true
	@docker ps -q --filter "name=$(APP_NAME)" | xargs -r docker stop || true
	@docker ps -aq --filter "name=$(APP_NAME)" | xargs -r docker rm || true
	@echo "$(COLOR_BLUE)Removing existing volume...$(COLOR_RESET)"
	@docker volume rm $(APP_NAME) 2>/dev/null || true
	@echo "$(COLOR_BLUE)Creating fresh volume...$(COLOR_RESET)"
	docker volume create $(APP_NAME)
	@echo "$(COLOR_BLUE)Running Docker container with fresh volume...$(COLOR_RESET)"
	docker run -d --rm --name $(APP_NAME) -p 8080:8080 -v $(APP_NAME):/app/data $(DOCKER_IMAGE):$(DOCKER_TAG)
	@echo "$(COLOR_GREEN)✓ Container started with fresh volume$(COLOR_RESET)"
	@echo "$(COLOR_CYAN)Application available at: http://localhost:8080$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Fresh volume created: $(APP_NAME) -> /app/data$(COLOR_RESET)"

.PHONY: docker-multiarch
docker-multiarch: ## Build Docker image for multiple architectures using buildx
	@echo "$(COLOR_BLUE)Building multi-architecture Docker image...$(COLOR_RESET)"
	docker buildx build --pull \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) -t $(DOCKER_IMAGE):latest . || { \
			echo "$(COLOR_YELLOW)⚠️  WARNING: Docker build cache unavailable (GitHub Actions cache error). Continuing without cache.$(COLOR_RESET)"; \
		}
	@echo "$(COLOR_GREEN)✓ Multi-architecture Docker image built$(COLOR_RESET)"

.PHONY: docker-multiarch-push
docker-multiarch-push: ## Build and push multi-architecture Docker image
	@echo "$(COLOR_BLUE)Building and pushing multi-architecture Docker image...$(COLOR_RESET)"
	docker buildx build --pull \
	--platform linux/amd64,linux/arm64 \
	--build-arg VERSION=$(VERSION) \
	--build-arg BUILD_TIME=$(BUILD_TIME) \
	--build-arg GIT_COMMIT=$(GIT_COMMIT) \
	-t $(DOCKER_IMAGE):$(DOCKER_TAG) -t $(DOCKER_IMAGE):latest --push . || { \
		echo "$(COLOR_YELLOW)⚠️  WARNING: Docker build cache unavailable (GitHub Actions cache error). Continuing without cache.$(COLOR_RESET)"; \
	}
	@echo "$(COLOR_GREEN)✓ Multi-architecture Docker image built and pushed$(COLOR_RESET)"

.PHONY: docker-local
docker-local: ## Build and push image locally with custom platforms
	@echo "$(COLOR_BLUE)Building Docker image locally (single platform, no push)...$(COLOR_RESET)"
	docker buildx build --pull \
		--platform linux/amd64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) -t $(DOCKER_IMAGE):latest \
		--load . || { \
			echo "$(COLOR_YELLOW)⚠️  WARNING: Docker build cache unavailable (GitHub Actions cache error). Continuing without cache.$(COLOR_RESET)"; \
		}
	@echo "$(COLOR_GREEN)✓ Local Docker image built and loaded locally$(COLOR_RESET)"

.PHONY: docker-local-push
docker-local-push: ## Build Docker image locally and push to registry
	@echo "$(COLOR_BLUE)Building Docker image locally (single platform, then pushing)...$(COLOR_RESET)"
	docker buildx build --pull \
		--platform linux/amd64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) -t $(DOCKER_IMAGE):latest \
		--load . || { \
			echo "$(COLOR_YELLOW)⚠️  WARNING: Docker build cache unavailable (GitHub Actions cache error). Continuing without cache.$(COLOR_RESET)"; \
		}
	@echo "$(COLOR_BLUE)Pushing Docker image to registry...$(COLOR_RESET)"
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE):latest
	@echo "$(COLOR_GREEN)✓ Local Docker image built and pushed$(COLOR_RESET)"

# For multi-platform builds and push, you must use the docker-container driver:
# docker buildx build --platform $(PLATFORMS) --push ...

.PHONY: docker-setup-buildx
docker-setup-buildx: ## Setup Docker buildx for multi-architecture builds
	@echo "$(COLOR_BLUE)Setting up Docker buildx...$(COLOR_RESET)"
	docker buildx create --use --name subtitle-manager-builder || true
	docker buildx inspect --bootstrap
	@echo "$(COLOR_GREEN)✓ Docker buildx configured$(COLOR_RESET)"

.PHONY: docker-push
docker-push: docker ## Build and push Docker image
	@echo "$(COLOR_BLUE)Pushing Docker image...$(COLOR_RESET)"
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE):latest
	@echo "$(COLOR_GREEN)✓ Docker image pushed$(COLOR_RESET)"

.PHONY: docker-clean
docker-clean: ## Clean Docker images and containers
	@echo "$(COLOR_BLUE)Cleaning Docker artifacts...$(COLOR_RESET)"
	docker image prune -f
	docker container prune -f
	@echo "$(COLOR_GREEN)✓ Docker cleaned$(COLOR_RESET)"

.PHONY: docker-test-amd64
docker-test-amd64: ## Build and test Docker image for AMD64
	@echo "$(COLOR_BLUE)Building Docker image for AMD64...$(COLOR_RESET)"
	docker buildx build \
		--platform linux/amd64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(DOCKER_IMAGE):amd64 --load .
	@echo "$(COLOR_GREEN)✓ AMD64 Docker image built$(COLOR_RESET)"

.PHONY: docker-test-arm64
docker-test-arm64: ## Build and test Docker image for ARM64
	@echo "$(COLOR_BLUE)Building Docker image for ARM64...$(COLOR_RESET)"
	docker buildx build \
		--platform linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(DOCKER_IMAGE):arm64 --load .
	@echo "$(COLOR_GREEN)✓ ARM64 Docker image built$(COLOR_RESET)"

#
# Fast Docker Build Targets
.PHONY: docker-fast docker-assets docker-push-assets docker-optimized docker-benchmark

# Build web assets as a separate Docker image
docker-assets:
	@echo "🏗️  Building web assets container..."
	docker build -f Dockerfile.assets -t $(DOCKER_IMAGE)/assets:latest .
	docker build -f Dockerfile.assets -t $(DOCKER_IMAGE)/assets:$(VERSION) .

# Push assets to registry
docker-push-assets: docker-assets
	@echo "📦 Pushing assets to registry..."
	docker push $(DOCKER_IMAGE)/assets:latest
	docker push $(DOCKER_IMAGE)/assets:$(VERSION)

# Fast build using pre-built assets
docker-fast:
	@echo "🚀 Fast Docker build using pre-built assets..."
	@if docker pull $(DOCKER_IMAGE)/assets:latest; then \
	docker build -f Dockerfile.hybrid -t $(DOCKER_IMAGE):$(VERSION)-fast .; \
	else \
	echo "⚠️  Pre-built assets not available, falling back to standard build"; \
	$(MAKE) docker-optimized; \
	fi

# Optimized multi-stage build
docker-optimized:
	@echo "🔧 Standard Docker build using Dockerfile.hybrid..."
	DOCKER_BUILDKIT=1 docker build \
		-f Dockerfile.hybrid \
		-t $(DOCKER_IMAGE):$(VERSION) \
		-t $(DOCKER_IMAGE):latest \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILDKIT_INLINE_CACHE=1 \
		--cache-from $(DOCKER_IMAGE):latest \
		.

# Benchmark build times
docker-benchmark:
		@echo "⏱️  Benchmarking Docker build methods..."
		@echo "Building with original Dockerfile..."
		@time docker build -f Dockerfile -t $(DOCKER_IMAGE):original . &>/dev/null || echo "Original build failed"
		@echo "Building with hybrid Dockerfile..."
		@time docker build -f Dockerfile.hybrid -t $(DOCKER_IMAGE):hybrid . &>/dev/null || echo "Hybrid build failed"
		@echo "Building with fast method (if assets available)..."
		@time docker build -f Dockerfile.hybrid -t $(DOCKER_IMAGE):fast . &>/dev/null || echo "Fast build failed (assets not available)"

#
# Protocol Buffers
#

.PHONY: proto-gen
proto-gen: ## Generate protobuf code using buf
	@echo "$(COLOR_BLUE)Generating protobuf code with buf...$(COLOR_RESET)"
	buf generate
	@echo "$(COLOR_GREEN)✓ Protobuf code generated$(COLOR_RESET)"

#
# Quality Assurance
#

.PHONY: qa
qa: go-fmt go-vet go-lint test webui-lint webui-test ## Run all quality assurance checks
	@echo "$(COLOR_GREEN)✓ All QA checks passed$(COLOR_RESET)"

.PHONY: qa-full
qa-full: qa test-race webui-test-e2e ## Run comprehensive QA including race detection and E2E
	@echo "$(COLOR_GREEN)✓ Full QA suite completed$(COLOR_RESET)"

.PHONY: pre-commit
pre-commit: go-fmt go-vet test ## Run pre-commit checks
	@echo "$(COLOR_GREEN)✓ Pre-commit checks passed$(COLOR_RESET)"

#
# Development Targets
#

.PHONY: dev
dev: build ## Build and run in development mode
	@echo "$(COLOR_BLUE)Starting development server...$(COLOR_RESET)"
	$(BINARY_PATH) web

.PHONY: dev-race
dev-race: build-race ## Build with race detection and run
	@echo "$(COLOR_BLUE)Starting development server with race detection...$(COLOR_RESET)"
	$(BINARY_PATH)-race web

.PHONY: dev-air
dev-air: install-air webui-build go-generate ## Start development server with live reloading using Air
	@echo "$(COLOR_BLUE)Starting Air for live reloading...$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Air will automatically rebuild and restart on Go file changes$(COLOR_RESET)"
	@echo "$(COLOR_CYAN)Version will be set to: dev-air$(COLOR_RESET)"
	@echo "$(COLOR_CYAN)Server will be available at: http://localhost:8080$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Press Ctrl+C to stop$(COLOR_RESET)"
	air

.PHONY: dev-air-fast
dev-air-fast: install-air go-generate ## Start Air quickly (assumes web UI already built)
	@echo "$(COLOR_YELLOW)⚡ Fast development mode - skipping initial web UI build$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Run 'make webui-build' manually if you need to rebuild web assets$(COLOR_RESET)"
	@echo "$(COLOR_BLUE)Starting Air for live reloading...$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Air will automatically rebuild and restart on Go file changes$(COLOR_RESET)"
	@echo "$(COLOR_CYAN)Version will be set to: dev-air$(COLOR_RESET)"
	@echo "$(COLOR_CYAN)Server will be available at: http://localhost:8080$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Press Ctrl+C to stop$(COLOR_RESET)"
	air

.PHONY: dev-air-verbose
dev-air-verbose: install-air ## Start Air with verbose logging
	@echo "$(COLOR_BLUE)Starting Air with verbose logging...$(COLOR_RESET)"
	air -d

.PHONY: install-air
install-air: ## Install or update Air for live reloading
	@echo "$(COLOR_BLUE)Installing/updating Air...$(COLOR_RESET)"
	@which air > /dev/null 2>&1 || { \
		echo "$(COLOR_YELLOW)Air not found, installing...$(COLOR_RESET)"; \
		go install github.com/cosmtrek/air@latest; \
	}
	@echo "$(COLOR_GREEN)✓ Air is ready$(COLOR_RESET)"

.PHONY: install
install: build ## Install binary to $GOPATH/bin
	@echo "$(COLOR_BLUE)Installing to GOPATH...$(COLOR_RESET)"
	go install $(GO_BUILD_FLAGS) .
	@echo "$(COLOR_GREEN)✓ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)$(COLOR_RESET)"

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "$(COLOR_BLUE)Installing development tools...$(COLOR_RESET)"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/cosmtrek/air@latest
	cd $(WEBUI_DIR) && npx playwright install
	@echo "$(COLOR_GREEN)✓ Development tools installed$(COLOR_RESET)"

.PHONY: fix
fix: go-fmt go-mod-tidy ## Fix common code issues (format, tidy modules)
	@echo "$(COLOR_GREEN)✓ Code fixes applied$(COLOR_RESET)"

.PHONY: fix-webui
fix-webui: webui-deps-clean ## Fix web UI dependency issues
	@echo "$(COLOR_GREEN)✓ Web UI dependencies fixed$(COLOR_RESET)"

.PHONY: quick-build
quick-build: webui-build go-generate ## Quick build without full clean (faster development)
	@echo "$(COLOR_BLUE)Quick building Go binary...$(COLOR_RESET)"
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) -o $(BINARY_PATH) .
	@echo "$(COLOR_GREEN)✓ Quick build complete$(COLOR_RESET)"

#
# Git and Release Management
#

.PHONY: git-hooks
git-hooks: ## Install git hooks
	@echo "$(COLOR_BLUE)Installing git hooks...$(COLOR_RESET)"
	./scripts/install-hooks.sh
	@echo "$(COLOR_GREEN)✓ Git hooks installed$(COLOR_RESET)"

.PHONY: version
version: ## Show version information
	@echo "$(COLOR_CYAN)Version Information:$(COLOR_RESET)"
	@echo "  App Name:    $(APP_NAME)"
	@echo "  Version:     $(VERSION)"
	@echo "  Build Time:  $(BUILD_TIME)"
	@echo "  Git Commit:  $(GIT_COMMIT)"
	@echo "  Go Version:  $(GO_VERSION)"

.PHONY: changelog
changelog: ## Generate changelog (requires git-chglog)
	@echo "$(COLOR_BLUE)Generating changelog...$(COLOR_RESET)"
	git-chglog -o CHANGELOG.md
	@echo "$(COLOR_GREEN)✓ Changelog generated$(COLOR_RESET)"

#
# Documentation
#

.PHONY: docs
docs: ## Generate documentation
	@echo "$(COLOR_BLUE)Generating documentation...$(COLOR_RESET)"
	go doc -all . > $(DOCS_DIR)/api.md
	@echo "$(COLOR_GREEN)✓ Documentation generated$(COLOR_RESET)"

.PHONY: docs-serve
docs-serve: ## Serve documentation locally
	@echo "$(COLOR_BLUE)Serving documentation...$(COLOR_RESET)"
	cd $(DOCS_DIR) && python3 -m http.server 8000

#
# Cleanup Targets
#

.PHONY: clean
clean: ## Clean build artifacts
	@echo "$(COLOR_BLUE)Cleaning build artifacts...$(COLOR_RESET)"
	rm -rf $(BIN_DIR)
	rm -rf $(DIST_DIR)
	rm -f coverage.out coverage.html
	rm -f subtitle-manager
	@echo "$(COLOR_GREEN)✓ Build artifacts cleaned$(COLOR_RESET)"

.PHONY: clean-air
clean-air: ## Clean Air temporary files and logs
	@echo "$(COLOR_BLUE)Cleaning Air artifacts...$(COLOR_RESET)"
	rm -rf tmp/
	rm -f build-errors.log
	@echo "$(COLOR_GREEN)✓ Air artifacts cleaned$(COLOR_RESET)"

.PHONY: clean-webui
clean-webui: ## Clean web UI artifacts
	@echo "$(COLOR_BLUE)Cleaning web UI artifacts...$(COLOR_RESET)"
	rm -rf $(DIST_DIR)
	rm -rf $(WEBUI_DIR)/node_modules
	@echo "$(COLOR_GREEN)✓ Web UI artifacts cleaned$(COLOR_RESET)"

.PHONY: clean-all
clean-all: clean clean-air clean-webui docker-clean ## Clean everything
	@echo "$(COLOR_GREEN)✓ Everything cleaned$(COLOR_RESET)"

#
# Maintenance Targets
#

.PHONY: update-deps
update-deps: ## Update all dependencies
	@echo "$(COLOR_BLUE)Updating Go dependencies...$(COLOR_RESET)"
	go get -u ./...
	go mod tidy
	@echo "$(COLOR_BLUE)Updating Web UI dependencies...$(COLOR_RESET)"
	cd $(WEBUI_DIR) && npm update
	@echo "$(COLOR_GREEN)✓ Dependencies updated$(COLOR_RESET)"

.PHONY: security-scan
security-scan: ## Run security scans
	@echo "$(COLOR_BLUE)Running security scans...$(COLOR_RESET)"
	go list -json -deps ./... | nancy sleuth
	cd $(WEBUI_DIR) && npm audit
	@echo "$(COLOR_GREEN)✓ Security scans completed$(COLOR_RESET)"

#
# CI/CD Targets
#

.PHONY: ci
ci: qa test-coverage docker ## Run CI pipeline
	@echo "$(COLOR_GREEN)✓ CI pipeline completed$(COLOR_RESET)"

.PHONY: cd
cd: ci docker-push ## Run CD pipeline
	@echo "$(COLOR_GREEN)✓ CD pipeline completed$(COLOR_RESET)"

.PHONY: release
release: qa-full build-all-platforms docker ## Prepare release
	@echo "$(COLOR_GREEN)✓ Release prepared$(COLOR_RESET)"

# Make sure intermediate files are not deleted
.PRECIOUS: $(WEBUI_DIR)/node_modules $(DIST_DIR)

# Default goal
.DEFAULT_GOAL := help

.PHONY: test-binary
test-binary: binary ## Test that the binary works correctly
	@echo "$(COLOR_BLUE)Testing binary functionality...$(COLOR_RESET)"
	$(BINARY_PATH) --help > /dev/null
	@echo "$(COLOR_GREEN)✓ Binary test passed$(COLOR_RESET)"

.PHONY: run-web
run-web: build ## Build and run the web server
	@echo "$(COLOR_BLUE)Starting web server...$(COLOR_RESET)"
	$(BINARY_PATH) web

.PHONY: check-cgo
check-cgo: ## Check if CGO is properly configured
	@echo "$(COLOR_BLUE)Checking CGO configuration...$(COLOR_RESET)"
	@echo "CGO_ENABLED=$(CGO_ENABLED)"
	@go env CGO_ENABLED
	@echo "$(COLOR_GREEN)✓ CGO check complete$(COLOR_RESET)"

.PHONY: webui-rebuild
webui-rebuild: ## Force rebuild web UI and regenerate Go assets
	@echo "$(COLOR_BLUE)🔄 Force rebuilding web UI...$(COLOR_RESET)"
	cd $(WEBUI_DIR) && rm -rf dist && npm run build
	@echo "$(COLOR_BLUE)Running go generate...$(COLOR_RESET)"
	go generate ./webui
	@echo "$(COLOR_GREEN)✓ Web UI rebuilt and Go assets regenerated$(COLOR_RESET)"

#
# E2E Testing Targets
#

.PHONY: end2end-tests
end2end-tests: setup-e2e-env build ## Start complete E2E testing environment
	@echo "$(COLOR_BLUE)Starting E2E testing environment...$(COLOR_RESET)"
	@./scripts/setup-e2e-environment.sh

.PHONY: setup-e2e-env
setup-e2e-env: ## Setup E2E environment configuration
	@echo "$(COLOR_BLUE)Setting up E2E environment...$(COLOR_RESET)"
	@if [ ! -f .env.local ]; then \
		echo "$(COLOR_YELLOW)⚠️  .env.local not found. Checking for OpenAI API key...$(COLOR_RESET)"; \
		if [ -z "$$OPENAI_API_KEY" ]; then \
			echo "$(COLOR_YELLOW)⚠️  No OpenAI API key found in environment.$(COLOR_RESET)"; \
			echo "$(COLOR_YELLOW)   Please either:$(COLOR_RESET)"; \
			echo "$(COLOR_YELLOW)   1. Copy .env.local.template to .env.local and add your API key$(COLOR_RESET)"; \
			echo "$(COLOR_YELLOW)   2. Set OPENAI_API_KEY environment variable$(COLOR_RESET)"; \
			echo "$(COLOR_YELLOW)   3. Continue without translation features for basic testing$(COLOR_RESET)"; \
			read -p "Continue without API key? (y/N): " continue_without_key; \
			if [ "$$continue_without_key" != "y" ] && [ "$$continue_without_key" != "Y" ]; then \
				echo "$(COLOR_RED)Exiting. Please configure API key and try again.$(COLOR_RESET)"; \
				exit 1; \
			fi; \
		else \
			echo "$(COLOR_GREEN)✓ Using OPENAI_API_KEY from environment$(COLOR_RESET)"; \
		fi; \
	else \
		echo "$(COLOR_GREEN)✓ Found .env.local configuration$(COLOR_RESET)"; \
		source .env.local; \
	fi
	@echo "$(COLOR_GREEN)✓ E2E environment setup complete$(COLOR_RESET)"

.PHONY: setup-e2e-testdir
setup-e2e-testdir: ## Create/verify E2E test directory structure
	@echo "$(COLOR_BLUE)Setting up E2E test directory structure...$(COLOR_RESET)"
	@mkdir -p testdir/{movies,tv,anime}
	@mkdir -p testdir/movies/{"The Matrix (1999)","Blade Runner 2049 (2017)","Interstellar (2014)",The_Dark_Knight}
	@mkdir -p testdir/tv/{"Breaking Bad (2008)","The Office (2005)","Game of Thrones (2011)",stranger_things_2016}
	@mkdir -p testdir/anime/{"Attack on Titan (2013)","Your Name (2016)","Spirited Away (2001)",one_piece-1999}
	@echo "$(COLOR_GREEN)✓ E2E test directory structure created$(COLOR_RESET)"

.PHONY: stop-e2e
stop-e2e: ## Stop the E2E testing environment
	@echo "$(COLOR_BLUE)Stopping E2E testing environment...$(COLOR_RESET)"
	@if [ -f /tmp/subtitle-manager-e2e.pid ]; then \
		PID=$$(cat /tmp/subtitle-manager-e2e.pid); \
		if kill -0 $$PID 2>/dev/null; then \
			kill $$PID; \
			echo "$(COLOR_GREEN)✓ E2E server stopped (PID: $$PID)$(COLOR_RESET)"; \
		else \
			echo "$(COLOR_YELLOW)⚠️  E2E server was not running$(COLOR_RESET)"; \
		fi; \
		rm -f /tmp/subtitle-manager-e2e.pid; \
	else \
		echo "$(COLOR_YELLOW)⚠️  No E2E server PID file found$(COLOR_RESET)"; \
		pkill -f subtitle-manager || true; \
	fi

.PHONY: clean-e2e
clean-e2e: stop-e2e ## Clean up E2E testing artifacts
	@echo "$(COLOR_BLUE)Cleaning E2E testing artifacts...$(COLOR_RESET)"
	@rm -f /tmp/subtitle-manager-e2e.log
	@rm -f /tmp/subtitle-manager-e2e.pid
	@echo "$(COLOR_GREEN)✓ E2E artifacts cleaned$(COLOR_RESET)"

.PHONY: status-e2e
status-e2e: ## Check E2E environment status
	@echo "$(COLOR_BLUE)Checking E2E environment status...$(COLOR_RESET)"
	@if [ -f /tmp/subtitle-manager-e2e.pid ]; then \
		PID=$$(cat /tmp/subtitle-manager-e2e.pid); \
		if kill -0 $$PID 2>/dev/null; then \
			echo "$(COLOR_GREEN)✓ E2E server running (PID: $$PID)$(COLOR_RESET)"; \
			echo "$(COLOR_BLUE)Web Interface: http://127.0.0.1:55327$(COLOR_RESET)"; \
			echo "$(COLOR_BLUE)Credentials: test/test123$(COLOR_RESET)"; \
		else \
			echo "$(COLOR_RED)✗ E2E server not running$(COLOR_RESET)"; \
		fi; \
	else \
		echo "$(COLOR_RED)✗ No E2E server running$(COLOR_RESET)"; \
	fi
	@if curl -s http://127.0.0.1:55327/health > /dev/null 2>&1; then \
		echo "$(COLOR_GREEN)✓ Health check passed$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_RED)✗ Health check failed$(COLOR_RESET)"; \
	fi

.PHONY: logs-e2e
logs-e2e: ## Show E2E testing logs
	@echo "$(COLOR_BLUE)Showing E2E testing logs...$(COLOR_RESET)"
	@if [ -f /tmp/subtitle-manager-e2e.log ]; then \
		tail -f /tmp/subtitle-manager-e2e.log; \
	else \
		echo "$(COLOR_YELLOW)⚠️  No E2E log file found$(COLOR_RESET)"; \
	fi
