default:
    @just --list

# Run all unit tests with race detection
test:
    go test -v -race -coverprofile=coverage.out ./...

# Install web UI dependencies
install-ui:
    cd web && npm ci

# Build the web UI into internal/ui/dist (embedded into the server binary)
build-ui:
    cd web && npm run build

# Run the web UI dev server with hot reload (proxies /api to `just dev-server`)
dev-ui:
    cd web && npm run dev

# Run the web UI unit tests and type check
test-ui:
    cd web && npm run type-check && npm test

# Build the server binary (including the web UI)
build: build-ui
    go build -v -o bin/arch-repo-server ./cmd/server

# Run the Go server in local development mode on :8080 (mock Kubernetes)
dev-server:
    go run ./cmd/server -dev -repo-dir /tmp/repo -listen :8080

# Run the embedded (built) UI against the dev server
dev: build-ui dev-server

# Format all Go source files
fmt:
    gofmt -w .

# Display statement coverage summary
coverage: test
    go tool cover -func=coverage.out

# Restart the pod in k3s/k8s (Deployment recreates it and pulls the latest image)
redeploy namespace="arch-repo" deployment="arch-repo-server":
    kubectl -n {{namespace}} delete pod -l app.kubernetes.io/name={{deployment}} --wait=true
    kubectl -n {{namespace}} rollout status deployment/{{deployment}} --timeout=180s
    kubectl -n {{namespace}} get pods -l app.kubernetes.io/name={{deployment}} -o wide
