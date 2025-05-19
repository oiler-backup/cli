PROJECT_NAME = oiler
GO_VERSION = 1.24
GO = go
GOFLAGS = -ldflags="-s -w"
BIN_DIR = bin
BIN_PATH = $(BIN_DIR)/$(PROJECT_NAME)

all: build

build:
	@echo "Building $(PROJECT_NAME)..."
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BIN_PATH) ./main.go
	@echo "Build completed. Binary located at $(BIN_PATH)"

deps:
	@echo "Installing dependencies..."
	@$(GO) mod tidy
	@echo "Dependencies installed."

test:
	@echo "Running tests..."
	@$(GO) test ./...
	@echo "Tests completed."

clean:
	@echo "Cleaning project..."
	@rm -rf $(BIN_DIR)
	@echo "Clean completed."

help:
	@echo "Usage:"
	@echo "  make build    - Build the project"
	@echo "  make deps     - Install dependencies"
	@echo "  make test     - Run tests"
	@echo "  make clean    - Clean the project"
	@echo "  make help     - Show this help message"

.PHONY: all build deps test clean help