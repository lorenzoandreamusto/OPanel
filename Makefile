BINARY_NAME=opaneld
BUILD_DIR=./bin
GO=go
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
NPM=npm

.PHONY: all build run clean test fmt vet frontend frontend-dev frontend-build deps

all: build

build: frontend-build
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/opaneld

run: build
	$(BUILD_DIR)/$(BINARY_NAME) server

clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR) static
	cd frontend && $(NPM) run clean 2>/dev/null || true
	$(GO) clean

test:
	$(GO) test ./... -v

fmt:
	$(GO) fmt ./...
	cd frontend && $(NPM) run typecheck 2>/dev/null || true

vet:
	$(GO) vet ./...

deps:
	$(GO) mod tidy
	$(GO) mod download
	cd frontend && $(NPM) install

frontend:
	cd frontend && $(NPM) install

frontend-dev:
	cd frontend && $(NPM) run dev

frontend-build:
	@echo "Building frontend..."
	cd frontend && $(NPM) install && $(NPM) run build
