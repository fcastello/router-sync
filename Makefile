# Router Sync Makefile

# Variables
BINARY_NAME=router-sync
BUILD_DIR=build
VERSION=$(shell cat VERSION)
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=$(shell date -u '+%Y-%m-%d_%H:%M:%S') -X main.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_UNIX=$(BINARY_NAME)_unix

# Version management
VERSION_FILE=VERSION
CHANGELOG_FILE=CHANGELOG.md

# Default target
all: clean build

# Build the application
build:
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) main.go

# Build for multiple platforms
build-all: clean
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 main.go
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 main.go
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 main.go

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -cover ./...

# Run tests with race detection
test-race:
	$(GOTEST) -v -race ./...

# Run benchmarks
bench:
	$(GOTEST) -v -bench=. ./...

# Download dependencies
deps:
	$(GOMOD) download

# Tidy dependencies
tidy:
	$(GOMOD) tidy

# Generate API documentation
docs:
	swag init

# Install development tools
install-tools:
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/git-chglog/git-chglog/cmd/git-chglog@latest
	go install github.com/goreleaser/goreleaser@latest

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	$(GOCMD) fmt ./...

# Vet code
vet:
	$(GOCMD) vet ./...

# Run all checks
check: fmt vet lint test

# Version management
version:
	@echo "Current version: $(VERSION)"

version-bump-patch:
	@echo "Bumping patch version..."
	@$(eval NEW_VERSION := $(shell echo $(VERSION) | awk -F. '{print $$1"."$$2"."$$3+1}'))
	@echo $(NEW_VERSION) > $(VERSION_FILE)
	@echo "Version bumped to: $(NEW_VERSION)"
	@git add $(VERSION_FILE)
	@git commit -m "chore: bump version to $(NEW_VERSION)"
	@git tag v$(NEW_VERSION)
	@echo "Tagged as v$(NEW_VERSION)"

version-bump-minor:
	@echo "Bumping minor version..."
	@$(eval NEW_VERSION := $(shell echo $(VERSION) | awk -F. '{print $$1"."$$2+1".0"}'))
	@echo $(NEW_VERSION) > $(VERSION_FILE)
	@echo "Version bumped to: $(NEW_VERSION)"
	@git add $(VERSION_FILE)
	@git commit -m "chore: bump version to $(NEW_VERSION)"
	@git tag v$(NEW_VERSION)
	@echo "Tagged as v$(NEW_VERSION)"

version-bump-major:
	@echo "Bumping major version..."
	@$(eval NEW_VERSION := $(shell echo $(VERSION) | awk -F. '{print $$1+1".0.0"}'))
	@echo $(NEW_VERSION) > $(VERSION_FILE)
	@echo "Version bumped to: $(NEW_VERSION)"
	@git add $(VERSION_FILE)
	@git commit -m "chore: bump version to $(NEW_VERSION)"
	@git tag v$(NEW_VERSION)
	@echo "Tagged as v$(NEW_VERSION)"

# Changelog generation
changelog:
	@echo "Generating changelog..."
	@git-chglog --output $(CHANGELOG_FILE) --next-tag v$(VERSION)
	@echo "Changelog generated: $(CHANGELOG_FILE)"

changelog-preview:
	@echo "Previewing changelog for next release..."
	@git-chglog --output - --next-tag v$(VERSION)

# Release management
release-prepare: check changelog
	@echo "Preparing release v$(VERSION)..."
	@echo "Please review the changelog and commit if ready:"
	@echo "  git add $(CHANGELOG_FILE)"
	@echo "  git commit -m 'docs: update changelog for v$(VERSION)'"
	@echo "  git push origin main"
	@echo "  git push origin v$(VERSION)"

release: release-prepare
	@echo "Creating release v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)/release
	@$(MAKE) build-all
	@cd $(BUILD_DIR) && tar -czf ../release/$(BINARY_NAME)-v$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	@cd $(BUILD_DIR) && tar -czf ../release/$(BINARY_NAME)-v$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	@echo "Release artifacts created in build/release/"
	@echo "To create GitHub release:"
	@echo "  gh release create v$(VERSION) --title 'Release v$(VERSION)' --notes-file $(CHANGELOG_FILE) build/release/*.tar.gz"

release-github: release
	@echo "Creating GitHub release v$(VERSION)..."
	@gh release create v$(VERSION) --title "Release v$(VERSION)" --notes-file $(CHANGELOG_FILE) build/release/*.tar.gz

# Create release
release-full: check changelog release-github
	@echo "Release v$(VERSION) completed successfully!"

# Install the binary
install: build
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

# Uninstall the binary
uninstall:
	sudo rm -f /usr/local/bin/$(BINARY_NAME)

# Run the application locally
run: build
	./$(BUILD_DIR)/$(BINARY_NAME) -config config.yaml

# Run with debug logging
run-debug: build
	./$(BUILD_DIR)/$(BINARY_NAME) -config config.yaml -log-level debug

# Docker build
docker-build:
	docker build -t router-sync:$(VERSION) .
	docker tag router-sync:$(VERSION) router-sync:latest

# Docker run
docker-run:
	docker run --rm -it --network host -v $(PWD)/config.yaml:/app/config.yaml router-sync:latest

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  build-all    - Build for multiple platforms"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage"
	@echo "  test-race    - Run tests with race detection"
	@echo "  bench        - Run benchmarks"
	@echo "  deps         - Download dependencies"
	@echo "  tidy         - Tidy dependencies"
	@echo "  docs         - Generate API documentation"
	@echo "  install-tools- Install development tools"
	@echo "  lint         - Run linter"
	@echo "  fmt          - Format code"
	@echo "  vet          - Vet code"
	@echo "  check        - Run all checks"
	@echo "  version      - Show current version"
	@echo "  version-bump-patch - Bump patch version"
	@echo "  version-bump-minor - Bump minor version"
	@echo "  version-bump-major - Bump major version"
	@echo "  changelog    - Generate changelog"
	@echo "  changelog-preview - Preview changelog"
	@echo "  release-prepare - Prepare release"
	@echo "  release      - Create release artifacts"
	@echo "  release-github - Create GitHub release"
	@echo "  release-full - Complete release workflow"
	@echo "  install      - Install binary"
	@echo "  uninstall    - Uninstall binary"
	@echo "  run          - Run locally"
	@echo "  run-debug    - Run with debug logging"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  help         - Show this help"

.PHONY: all build build-all clean test test-coverage test-race bench deps tidy docs install-tools lint fmt vet check version version-bump-patch version-bump-minor version-bump-major changelog changelog-preview release-prepare release release-github release-full install uninstall run run-debug docker-build docker-run help 
