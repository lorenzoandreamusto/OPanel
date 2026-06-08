BINARY_NAME=opaneld
BUILD_DIR=./bin
GO=go
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: all build run clean test fmt vet

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/opaneld

run: build
	$(BUILD_DIR)/$(BINARY_NAME) server

clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	$(GO) clean

test:
	$(GO) test ./... -v

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

deps:
	$(GO) mod tidy
	$(GO) mod download
