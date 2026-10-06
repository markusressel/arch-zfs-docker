# AGENTS.md — arch-zfs-docker

Autonomous agent guide and engineering handbook for `arch-zfs-docker`.

## Project Overview

Modern Go web service, package catalog dashboard, upstream kernel scheduler, and Kubernetes build runner for building and serving Arch Linux `zfs-linux` and `zfs-utils` packages.

Linux-based, containerized, designed for Kubernetes (k3s/k8s) environments and standalone local Docker execution.

## Quick Reference Commands

| Task | Command |
|---|---|
| Build check (Go) | `go build ./... && go vet ./...` |
| Run all tests | `go test -v ./...` |
| Local server dev run | `go run ./cmd/server -dev -repo-dir /tmp/repo -listen :8080` |
| Docker build | `docker build -t arch-zfs-docker .` |
| Format code | `gofmt -w .` |
| Check git diff | `git diff` |

## Repository Architecture

```text
cmd/
  server/
    main.go                   # Main service entrypoint
internal/
  config/
    config.go                 # Environment flags & runtime configuration
  k8s/
    client.go                 # In-cluster & out-of-cluster Kubernetes API client
    kubeconfig.go             # Local kubeconfig parser for out-of-cluster dev
    mock.go                   # Mock client for development without K8s access
    types.go                  # Job definitions and K8sClient interface
  kernel/
    kernel.go                 # Kernel version normalization & Arch Linux Archive version listing
  repo/
    indexer.go                # Filesystem package scanner, checksums & metadata parser
    types.go                  # PackageInfo & RepoSummary data structures
  scheduler/
    scheduler.go              # Upstream Arch Linux API poller & automatic build trigger
  server/
    server.go                 # HTTP server, REST API, SSE streaming & pacman file routing
  ui/
    embed.go                  # Go embed filesystem wrapper
    static/
      index.html              # Dark-mode single page dashboard
      style.css               # Modern clean design stylesheet
      app.js                  # Frontend client (search, SSE log viewer, triggers)
builder/                      # Standalone local Docker build scripts (legacy/workstation)
  Dockerfile
  populate-package-repository.sh
  scripts/
deploy/
  k8s/                        # Kubernetes deployment manifests
    rbac.yaml                 # ServiceAccount, Role, and RoleBinding
    deployment.yaml           # App Deployment
    service.yaml              # ClusterIP Service
    ingress.yaml              # Ingress with SSE buffering disabled
.github/
  workflows/                  # GitHub Actions CI/CD to build and publish to GHCR
```

## Architectural Guidelines

1. **Clean Pacman Static Serving**:
   - `/$repo/$arch/*.pkg.tar*` files are served with `Cache-Control: public, max-age=2592000, immutable`.
   - `/$repo/$arch/*.db*` and `*.files*` databases are served with `Cache-Control: no-store, no-cache, must-revalidate, max-age=0`.
2. **Kubernetes Isolation**:
   - In-cluster execution uses the dedicated `ServiceAccount` and standard REST API calls (`/apis/batch/v1/...`).
   - Log streaming consumes `/api/v1/namespaces/{ns}/pods/{pod}/log?follow=true` and pipes to HTTP Server-Sent Events (SSE).
   - SSE client disconnects must immediately close upstream Kubernetes HTTP responses via context propagation.
3. **High Performance Log Rendering**:
   - Web frontend must batch log chunks with `requestAnimationFrame` and maintain a bounded terminal scrollback to prevent browser freezing during high-throughput compilation output.
4. **Upstream Polling**:
   - Background polling queries `https://archlinux.org/packages/.../json/` directly without spawning containers or running `pacman` locally.
   - Jobs are triggered only when a version mismatch is detected between upstream and the indexed repository.

## Testing & Quality Expectations

- Unit tests live alongside packages (`*_test.go`).
- Always run `go test -v ./...` and `gofmt -l .` before committing.
- Do not check in private hostnames, cluster IP addresses, local absolute user paths, or hardware-specific machine names into repository files or documentation.
