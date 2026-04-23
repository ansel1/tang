set unstable

# Go related variables
binary_name := "tang"
gobase := justfile_directory()
gobin := gobase / "bin"
GO_TEST := env("GO_TEST", if which("tang") != "" { "tang test" } else { "go test" })

# Default: show available recipes
default:
    @just --list

# Run tests and build the binary
all: tidy fmt build test lint

# Build the binary
build:
    go build -o {{ gobin }}/{{ binary_name }} .

# Run the application
run:
    go run .

# Run tests
test:
    {{ GO_TEST }} ./...

# Clean build cache and binary
clean:
    go clean
    rm -rf {{ gobin }}

# Format code
fmt:
    go fmt ./...

# Lint code using golangci-lint
lint:
    golangci-lint run

# Tidy dependencies
tidy:
    go mod tidy

# Install the binary
install:
    go build -o $(go env GOPATH)/bin/{{ binary_name }} .
