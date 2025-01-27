# List all available recipes
default:
    @just --list

# Build the project
build:
    go build -o bin/sage

# Run all tests
test:
    go test -v ./...

# Run tests with coverage
test-coverage:
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out

# Run linter
lint:
    golangci-lint run

# Clean build artifacts
clean:
    rm -rf bin
    rm -f coverage.out

# Install development dependencies
setup:
    go mod download
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Format code
fmt:
    go fmt ./...

# Run the application
run *ARGS:
    go run main.go {{ARGS}}