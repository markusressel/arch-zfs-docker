# Arch Linux ZFS Package Repository & Builder

Modern Go-based web service, package catalog dashboard, and Kubernetes job runner for building and serving Arch Linux `zfs-linux` and `zfs-utils` packages.

> **Architecture & Deployment:** The service is designed to run inside a **Kubernetes cluster** (e.g. k3s). It uses the Kubernetes API to orchestrate build Jobs on dedicated nodes, stream container logs live via Server-Sent Events (SSE), and serve pacman repository files from a shared PersistentVolume (PVC). See [**DEPLOYMENT.md**](DEPLOYMENT.md) for step-by-step cluster setup instructions. For local testing outside of a cluster, a mock client mode (`-dev`) is included.

---

## Deployment

Refer to [**DEPLOYMENT.md**](DEPLOYMENT.md) for complete instructions on creating the namespace, persistent storage (PVC), RBAC permissions, deployment, service, and ingress in Kubernetes.

---

## Features

- **Kubernetes-Native Build Runner:** Automatically detects new upstream Arch kernels or triggers builds on demand, spawning isolated builder Jobs in the cluster.
- **Pacman Repository Server:** Serves packages and databases (`/$repo/$arch/*`) with optimal cache-control headers directly from a PVC.
- **Web Dashboard:** Interactive UI showing available packages, sizes, checksums, and kernel versions.
- **On-Demand Build Trigger & Upstream Sync:** Start builds directly from the web UI, trigger upstream checks, or configure auto-check intervals dynamically.
- **Kernel/ZFS Compatibility Tracking:** Warns when the kernel is newer than the packaged OpenZFS release supports (`Linux-Maximum` from its `META`), learns from failed builds, pauses futile retries until a new OpenZFS release lands, and links related OpenZFS GitHub issues.
- **Real-Time Log Streaming:** Stream build compilation logs in real-time via Server-Sent Events (SSE) with smart autoscrolling.
- **Persistent Build Caches:** PVC caching for pacman dependencies, OpenZFS source tarballs, and pre-built `zfs-utils` packages for fast builds.
- **Local Docker Builder Preserved:** Standalone local Docker build scripts remain available under [`builder/`](builder/).

---

## Project Structure

```text
├── cmd/
│   └── server/          # Main HTTP server entrypoint
├── internal/
│   ├── config/          # Environment configuration
│   ├── k8s/             # Kubernetes client (Job creation & SSE log streaming)
│   ├── repo/            # Filesystem package indexer & metadata parser
│   ├── server/          # HTTP server, REST API, & pacman file handlers
│   └── ui/              # Go embed of the built web UI
├── web/                 # Vue 3 + TypeScript frontend (Vite), built into internal/ui/dist
├── builder/             # Standalone local Docker build scripts (legacy workflow)
│   ├── Dockerfile
│   ├── populate-package-repository.sh
│   └── scripts/
├── deploy/
│   └── k8s/             # Kubernetes manifests (RBAC, Deployment, Service, Ingress)
├── Dockerfile           # Multi-stage container build for the Go web service
└── go.mod
```

---

## Local Development

Run the web service locally against your local package directory:

```bash
just dev        # server (builds run in the local Docker engine, repo in .dev/repo) + web UI with hot reload on :5173
just dev-embedded # builds the UI and serves it embedded from :8080 (like production)
just dev-mock   # fake build jobs, nothing is built
# or, without just, with a mock K8s client and a custom directory
go run ./cmd/server -repo-dir ~/.custom/zfs -dev -listen :8080
```

The web UI (`web/`, Vue 3 + TypeScript + Vite) is embedded into the Go binary, so build it once first:

```bash
just install-ui && just build-ui
```

`just dev` serves the web UI with hot reload at [http://localhost:5173/ui/](http://localhost:5173/ui/) (Vite proxies `/api` to the Go server on `:8080`). The UI on `:8080` is the last `just build-ui` output and is not live.

### Building without Kubernetes (local Docker)

Without Kubernetes credentials the service runs builds as containers on the local Docker engine (`BUILD_RUNNER=auto`; force it with `-build-runner docker`). Build containers use the `archlinux:base-devel` image and the same scripts as the Kubernetes jobs:

```bash
go run ./cmd/server -repo-dir ~/.custom/zfs -listen :8080
```

The repository directory is bind-mounted at `/repo` of each build container, so it must be a path the Docker host can see. When the service itself runs in a container with `/var/run/docker.sock` mounted, set `DOCKER_REPO_MOUNT` to the host path (or named volume) behind its `/repo`. `-dev` still uses the mock client and builds nothing.

Run tests:
```bash
go test -v ./...
just test-ui   # type check + web UI unit tests
```

---

## Client Setup (Arch Linux)

Add this repository to `/etc/pacman.conf` on any machine:

```ini
[zfslocal]
SigLevel = Optional TrustAll
Server = https://<your-domain-or-host>/$repo/$arch
```

Install packages:
```bash
sudo pacman -Sy
sudo pacman -S zfs-linux zfs-utils
```

---

## Local Docker Build (Standalone)

To build packages locally on your workstation using Docker without Kubernetes:

```bash
cd builder
REPOSITORY_NAME="zfslocal" REPOSITORY_PATH=~/.custom/zfs ./populate-package-repository.sh
```

