.PHONY: build run test clean tidy

# Binary name
BINARY_NAME=ext-kyma-mcp

# Build directory
BUILD_DIR=bin

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/main.go

# Run the application
run:
	$(GORUN) ./cmd/main.go

# Run tests
test:
	$(GOTEST) -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

# Tidy dependencies
tidy:
	$(GOMOD) tidy

# Format code
fmt:
	$(GOFMT) ./...

# Download dependencies
deps:
	$(GOMOD) download

# Verify dependencies
verify:
	$(GOMOD) verify

# Update dependencies
update:
	$(GOMOD) tidy
	$(GOMOD) verify
