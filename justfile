# List all available recipes
default:
    @just --list

version := "0.0.0-dev-" + `date "+%H.%M.%d.%m.%Y"`
ldflags := "-X 'github.com/crazywolf132/sage/cmd.Version=" + version + "'"

# Build the project (release build)
build:
    go build -o bin/sage .

# Build the development version with embedded version info
# The version format is: {major}.{minor}.{patch}-dev-{hour}.{minute}.{day}.{month}.{year}
# Example: 0.0.0-dev-15.04.02.01.2006
build-dev:
    go build -ldflags='{{ldflags}}' -o bin/sage .

# Run all tests
test:
    do go test ./...

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