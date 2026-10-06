# Arch Linux ZFS Package Repository & Builder

Modern Go-based web service, package catalog dashboard, and Kubernetes job runner for building and serving Arch Linux `zfs-linux` and `zfs-utils` packages.

---

## Features

- **Pacman Repository Server:** Serves packages and databases (`/$repo/$arch/*`) with optimal cache-control headers.
- **Web Dashboard:** Interactive UI showing available packages, sizes, checksums, and kernel versions.
- **On-Demand Build Trigger:** Start build jobs directly from the web UI with custom kernel versions or LTS variant.
- **Real-Time Log Streaming:** Stream build compilation logs in real-time via Server-Sent Events (SSE).
- **Cluster Native (k3s):** Runs builds as isolated container jobs on high-performance nodes.
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
│   └── ui/              # Embedded frontend dashboard (HTML, CSS, JS)
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
# Run with mock K8s client and local custom directory
go run ./cmd/server -repo-dir ~/.custom/zfs -dev -listen :8080
```

Open [http://localhost:8080](http://localhost:8080) in your browser.

Run tests:
```bash
go test -v ./...
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
REPOSITORY_NAME="zfslocal" REPOSITORY_PATH=/home/markus/.custom/zfs ./populate-package-repository.sh
```

