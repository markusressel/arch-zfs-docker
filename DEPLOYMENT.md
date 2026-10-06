# Deployment Guide

This guide details how to deploy the Arch Linux ZFS Package Repository & Builder service in a Kubernetes cluster (e.g., [k3s](https://k3s.io/), [microk8s](https://microk8s.io/), or standard Kubernetes).

---

## Architecture Overview

The system runs natively within a Kubernetes cluster:

```text
               +-------------------------------------------------+
               |                   Kubernetes                    |
               |                                                 |
Clients ------>| Ingress -> Service -> arch-repo-server (Go)    |
(Pacman & UI)  |                              |                  |
               |                              | (Spawns Jobs)    |
               |                              v                  |
               |                   zfs-build-<timestamp> (Pod)   |
               |                              |                  |
               +------------------------------|------------------+
                                              |
                                              v
                              +-------------------------------+
                              |    PersistentVolume (PVC)     |
                              |  /repo/zfslocal/x86_64        |
                              |  /repo/cache/pacman           |
                              |  /repo/cache/sources          |
                              |  /repo/logs                   |
                              +-------------------------------+
```

- **`arch-repo-server`**: Runs 24/7 as a lightweight Deployment. It serves the pacman repository files, hosts the web dashboard, checks upstream Arch kernel updates, and streams build logs via Server-Sent Events (SSE).
- **Builder Jobs**: When a kernel update is detected or manually triggered, the service launches an ephemeral Kubernetes `Job` (`zfs-build-<id>`) that compiles ZFS against the target kernel.
- **Persistent Volume**: Both the server and ephemeral builder pods share a single PVC (`arch-repo-data`) mounted at `/repo`. This ensures packages, repository databases, download caches, and logs persist across builds.

---

## Prerequisites

- A running Kubernetes cluster (v1.26+)
- `kubectl` configured with cluster admin permissions
- A default `StorageClass` supporting `ReadWriteOnce` (or `ReadWriteMany`)
- An Ingress controller (e.g. `ingress-nginx`, `traefik`)
- Optional: `cert-manager` for automatic Let's Encrypt TLS certificates

---

## Step-by-Step Deployment

All manifests are located in [`deploy/k8s/`](deploy/k8s/).

### 1. Create Namespace

```bash
kubectl create namespace arch-repo
```

### 2. Create Storage (PVC)

Verify [`deploy/k8s/pvc.yaml`](deploy/k8s/pvc.yaml) matches your desired storage size and storage class (default: 20Gi):

```bash
kubectl apply -f deploy/k8s/pvc.yaml
```

> **Note:** If your cluster uses a specific `StorageClass`, add `storageClassName: <class-name>` under `spec` in `deploy/k8s/pvc.yaml`.

### 3. Grant RBAC Permissions

The web server needs permissions within the `arch-repo` namespace to create batch jobs and stream pod logs:

```bash
kubectl apply -f deploy/k8s/rbac.yaml
```

This creates:
- `ServiceAccount`: `arch-repo-server`
- `Role`: `arch-repo-server-role` (grants `get`, `list`, `watch`, `create`, `delete` on `batch/jobs` and `pods/log`)
- `RoleBinding`: `arch-repo-server-binding`

### 4. Configure & Deploy Server

Review [`deploy/k8s/deployment.yaml`](deploy/k8s/deployment.yaml) and adjust environment variables if desired:

| Variable | Default | Description |
|---|---|---|
| `LISTEN_ADDR` | `:8080` | Internal server HTTP port |
| `REPO_DIR` | `/repo` | Root directory of repository volume |
| `REPO_NAME` | `zfslocal` | Pacman repository name |
| `K8S_NAMESPACE` | `arch-repo` | Kubernetes namespace for builder jobs |
| `BUILD_NODE` | _(empty)_ | Node name (`kubernetes.io/hostname`) to constrain build jobs to (optional) |
| `AUTO_CHECK_INTERVAL` | `6h` | Frequency to check Arch Linux for new kernel updates (`0` to disable) |

Deploy the service:

```bash
kubectl apply -f deploy/k8s/deployment.yaml
```

### 5. Expose Service

Create internal `ClusterIP` Service:

```bash
kubectl apply -f deploy/k8s/service.yaml
```

### 6. Configure Ingress & Domain

Edit [`deploy/k8s/ingress.yaml`](deploy/k8s/ingress.yaml) to replace the placeholder domain (`zfs.example.com`) with your actual domain name and adjust your `ingressClassName`:

```yaml
spec:
  ingressClassName: "nginx" # Or "traefik", etc.
  tls:
  - hosts:
    - your-domain.com
    secretName: arch-repo-tls
  rules:
  - host: your-domain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: arch-repo-service
            port:
              number: 80
```

Apply the Ingress:

```bash
kubectl apply -f deploy/k8s/ingress.yaml
```

---

## Dedicated Build Nodes (Optional)

Building kernel modules requires significant CPU and memory. You can dedicate a powerful worker node in your cluster to run builds:

1. Label your target node:
   ```bash
   kubectl label node <node-name> build-worker=true
   ```
2. Set the `BUILD_NODE` environment variable in `deploy/k8s/deployment.yaml`:
   ```yaml
   - name: BUILD_NODE
     value: "<node-name>"
   ```
   *(You can also configure this dynamically in the Web UI under Settings without restarting the deployment).*

---

## Verifying the Deployment

1. Check that the server pod is running:
   ```bash
   kubectl -n arch-repo get pods
   ```
2. View startup logs:
   ```bash
   kubectl -n arch-repo logs -l app.kubernetes.io/name=arch-repo-server -f
   ```
3. Open `https://your-domain.com` in your browser. You will see the repository dashboard.
4. Click **"Check Now"** or **"Start Build Job"** to trigger your first ZFS package build and verify the live SSE log terminal.

---

## Pacman Client Configuration

On Arch Linux client machines, add the repository to `/etc/pacman.conf`:

```ini
[zfslocal]
SigLevel = Optional TrustAll
Server = https://your-domain.com/$repo/$arch
```

Synchronize pacman and install:

```bash
sudo pacman -Sy
sudo pacman -S zfs-linux zfs-utils
```
