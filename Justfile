default:
    @just --list

# Run all unit tests with race detection
test:
    go test -v -race -coverprofile=coverage.out ./...

# Build the server binary
build:
    go build -v -o bin/arch-repo-server ./cmd/server

# Run server in local development mode
dev:
    go run ./cmd/server -dev

# Format all Go source files
fmt:
    gofmt -w .

# Display statement coverage summary
coverage: test
    go tool cover -func=coverage.out
