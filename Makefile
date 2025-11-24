# Binary name
BINARY_NAME=tang

# Go related variables.
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin
GOFILES=$(shell find . -type f -name '*.go' -not -path "./vendor/*")

# Make is verbose in Linux. Make it silent.
MAKEFLAGS += --silent

.PHONY: all build run test clean fmt lint help

all: test build ## Run tests and build the binary

build: ## Build the binary
	@echo "  >  Building binary..."
	@go build -o $(GOBIN)/$(BINARY_NAME) .

run: ## Run the application
	@go run .

test: ## Run tests
	@echo "  >  Running tests..."
	@go test ./...

clean: ## Clean build cache and binary
	@echo "  >  Cleaning build cache"
	@go clean
	@rm -rf $(GOBIN)

fmt: ## Format code
	@echo "  >  Formatting code..."
	@go fmt ./...

lint: ## Lint code using golangci-lint
	@echo "  >  Linting code..."
	@golangci-lint run

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

install: ## Install the binary
	@echo "  >  Installing binary..."
	@go build -o $(shell go env GOPATH)/bin/$(BINARY_NAME) .
