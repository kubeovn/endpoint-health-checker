SHELL = /bin/bash

# Build configuration
COMMIT = git-$(shell git rev-parse --short HEAD)
DATE = $(shell date +"%Y-%m-%d_%H:%M:%S")
VERSION ?= $(shell cat VERSION | tr -d '\n')
REGISTRY = docker.io/endpoint_health_checker
RELEASE_TAG = $(VERSION)
DEV_TAG = dev

# Go build flags
GOLDFLAGS = -extldflags '-z now' -X main.COMMIT=$(COMMIT) -X main.VERSION=$(RELEASE_TAG) -X main.BUILDDATE=$(DATE)
ifdef DEBUG
GO_BUILD_FLAGS = -ldflags "$(GOLDFLAGS)"
else
GO_BUILD_FLAGS = -trimpath -ldflags "-w -s $(GOLDFLAGS)"
endif

# Docker build configuration
DOCKERFILE = dist/images/Dockerfile
BUILD_CONTEXT = .

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf dist/images/endpoint_health_checker
	rm -f *.tar
	rm -f kube-ovn*.tar
	rm -f image*.tar

.PHONY: go-mod-tidy
go-mod-tidy: ## Run go mod tidy
	go mod tidy
	go mod download

.PHONY: build-go
build-go: go-mod-tidy ## Build Go binary
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(CURDIR)/dist/images/endpoint_health_checker ./main.go

.PHONY: build
build: build-go ## Build the project (alias for build-go)

.PHONY: build-debug
build-debug: ## Build with debug symbols
	@DEBUG=1 $(MAKE) build-go

.PHONY: build-image
build-image: build ## Build Docker image
	docker build -t $(REGISTRY):$(RELEASE_TAG) --build-arg VERSION=$(RELEASE_TAG) -f $(DOCKERFILE) $(BUILD_CONTEXT)

.PHONY: build-dev-image
build-dev-image: build ## Build development Docker image
	docker build -t $(REGISTRY):$(DEV_TAG) --build-arg VERSION=$(RELEASE_TAG) -f $(DOCKERFILE) $(BUILD_CONTEXT)

.PHONY: build-image-debug
build-image-debug: build-debug ## Build debug Docker image
	docker build -t $(REGISTRY):$(RELEASE_TAG)-debug --build-arg VERSION=$(RELEASE_TAG) -f $(DOCKERFILE) $(BUILD_CONTEXT)

.PHONY: lint
lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	golangci-lint run --timeout=10m

.PHONY: lint-fix
lint-fix: ## Run golangci-lint with auto-fix
	@echo "Running golangci-lint with auto-fix..."
	golangci-lint run --fix --timeout=10m

.PHONY: test
test: ## Run tests
	go test -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: security-scan
security-scan: ## Run security scan with gosec
	@echo "Running gosec security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found, skipping security scan. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi

.PHONY: verify
verify: lint test security-scan ## Run all verification steps (lint, test, security)

.PHONY: install-chart
install-chart: build-image ## Install chart using Helm
	helm install endpoint-health-checker ./chart --namespace kube-system --wait

.PHONY: upgrade-chart
upgrade-chart: build-image ## Upgrade chart using Helm
	helm upgrade endpoint-health-checker ./chart --namespace kube-system --wait

.PHONY: uninstall-chart
uninstall-chart: ## Uninstall chart using Helm
	helm uninstall endpoint-health-checker --namespace kube-system || true

.PHONY: kind-init
kind-init: ## Initialize Kind cluster
	kind create cluster --name test-cluster --wait 30s

.PHONY: kind-load-image
kind-load-image: build-image ## Load Docker image into Kind cluster
	kind load docker-image $(REGISTRY):$(RELEASE_TAG) --name test-cluster

.PHONY: kind-install
kind-install: kind-load-image install-chart ## Load image and install chart in Kind

.PHONY: kind-test
kind-test: kind-install ## Run Kind-based E2E test
	@echo "Running Kind E2E tests..."
	kubectl wait --for=condition=ready pod -l app=endpoint-health-checker -n kube-system --timeout=300s
	kubectl get pods -n kube-system -l app=endpoint-health-checker

.PHONY: kind-cleanup
kind-cleanup: ## Cleanup Kind cluster
	kind delete cluster --name test-cluster || true

.PHONY: ci
ci: verify build ## Run CI pipeline locally

.PHONY: local-dev
local-dev: ## Setup local development environment
	@echo "Setting up local development environment..."
	@DEBUG=1 $(MAKE) build-go
	$(MAKE) kind-init
	$(MAKE) kind-install

# Docker operations
.PHONY: docker-push
docker-push: build-image ## Push Docker image to registry
	docker push $(REGISTRY):$(RELEASE_TAG)

.PHONY: docker-tag-latest
docker-tag-latest: build-image ## Tag image as latest
	docker tag $(REGISTRY):$(RELEASE_TAG) $(REGISTRY):latest

.PHONY: docker-push-latest
docker-push-latest: docker-tag-latest ## Push latest Docker image
	docker push $(REGISTRY):latest

.PHONY: docker-save
docker-save: build-image ## Save Docker image to tar
	docker save $(REGISTRY):$(RELEASE_TAG) -o endpoint-health-checker-$(RELEASE_TAG).tar

# Development helpers
.PHONY: fmt
fmt: ## Format Go code
	go fmt ./...
	goimports -w .

.PHONY: mod-verify
mod-verify: ## Verify go.mod and go.sum consistency
	go mod verify

.PHONY: deps-update
deps-update: ## Update dependencies
	go get -u ./...
	go mod tidy

# Documentation
.PHONY: docs-serve
docs-serve: ## Serve documentation locally (if docs exist)
	@if [ -d "docs" ]; then \
		cd docs && python3 -m http.server 8000; \
	else \
		echo "No docs directory found"; \
	fi

# Release helpers
.PHONY: release-check
release-check: verify ## Perform release checks
	@echo "Performing release checks..."
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Date: $(DATE)"

.PHONY: release
release: release-check build-image docker-save ## Create release artifacts
	@echo "Release artifacts created successfully"
	@echo "Docker image: $(REGISTRY):$(RELEASE_TAG)"
	@echo "Tar file: endpoint-health-checker-$(RELEASE_TAG).tar"