# Binary name
BINARY_NAME=tang

# Go related variables.
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin

# Make is verbose in Linux. Make it silent.
MAKEFLAGS += --silent

.PHONY: all build run test clean fmt lint tidy help

all: tidy fmt build test lint ## Run tests and build the binary

build: ## Build the binary
	go build -o $(GOBIN)/$(BINARY_NAME) .

run: ## Run the application
	go run .

test: ## Run tests
	go test ./...

clean: ## Clean build cache and binary
	go clean
	rm -rf $(GOBIN)

fmt: ## Format code
	go fmt ./...

lint: ## Lint code using golangci-lint
	golangci-lint run

tidy: ## Tidy dependencies
	go mod tidy

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

install: ## Install the binary
	go build -o $(shell go env GOPATH)/bin/$(BINARY_NAME) .
