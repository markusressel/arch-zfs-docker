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

# Run only the web UI dev server with hot reload (proxies /api to `just dev-server`)
dev-ui:
    cd web && npm run dev

# Run the web UI unit tests and type check
test-ui:
    cd web && npm run type-check && npm test

# Build the server binary (including the web UI)
build: build-ui
    go build -v -o bin/arch-repo-server ./cmd/server

# Run the Go server on :8080; builds run as containers in the local Docker engine
# (the runner is forced so a kubeconfig on this machine is never used for dev builds)
dev-server:
    mkdir -p .dev/repo
    go run ./cmd/server -build-runner docker -repo-dir {{justfile_directory()}}/.dev/repo -listen :8080

# Run the Go server in mock mode: fake build jobs, nothing is built
dev-mock:
    go run ./cmd/server -dev -repo-dir /tmp/repo -listen :8080

# Full dev environment: Go server (real builds in local Docker) plus the web UI with hot reload.
# Open http://localhost:5173/ui/ (the UI on :8080 is the last `just build-ui`, not live).
dev:
    #!/usr/bin/env bash
    set -euo pipefail
    trap 'kill 0' EXIT
    echo "==> Web UI with hot reload: http://localhost:5173/ui/"
    (cd web && npm run dev -- --strictPort) &
    just dev-server

# Build the UI and run it embedded in the Go server (what production serves, no hot reload)
dev-embedded: build-ui dev-server

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
