# Makefile
.PHONY: build clean test lint run help

BINARY_NAME=gcp-auditor
BUILD_DIR=bin

help:
	@echo "Available commands:"
	@echo "  make build    - Build the application"
	@echo "  make clean    - Clean build artifacts"
	@echo "  make test     - Run tests"
	@echo "  make lint     - Run linter"
	@echo "  make run      - Run the application"
	@echo "  make all      - Clean, build, and test"

build:
	@echo "Building..."
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) main.go

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

test:
	@echo "Running tests..."
	@go test -v ./...

lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint is not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

run:
	@echo "Running..."
	@go run main.go audit --verbose

all: clean build test



# --- Previous Makefile ---

# .PHONY: build clean test lint run

# BINARY_NAME=gcp-auditor
# BINARY_UNIX=$(BINARY_NAME)_unix

# build:
# 	go build -o bin/$(BINARY_NAME) main.go

# clean:
# 	go clean
# 	rm -f bin/$(BINARY_NAME)
# 	rm -f bin/$(BINARY_UNIX)

# test:
# 	go test ./...

# lint:
# 	golangci-lint run

# run:
# 	go run main.go

# # Cross compilation
# build-linux:
# 	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY_UNIX) main.go