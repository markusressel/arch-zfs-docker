# Deploy in Kubernetes

The server runs as a Deployment. It starts each build as a Job in its own namespace, and both share one volume
mounted at `/repo`:

```text
pacman, browser ──> Ingress ──> Service ──> arch-repo-server ──(starts)──> Job zfs-build-<id>
                                                   │                              │
                                                   └──────── PVC arch-repo-data ──┘
                                                       /repo/zfslocal/x86_64   packages and database
                                                       /repo/cache/            pacman packages, sources
                                                       /repo/logs/             build logs
                                                       /repo/compat.json       learned incompatibilities
```

The manifests are in
[`deploy/k8s/`](https://github.com/markusressel/arch-zfs-docker/tree/main/deploy/k8s).

## Requirements

- Kubernetes 1.26 or newer, for example k3s.
- A StorageClass for a 20 GiB volume. `ReadWriteOnce` works when the server and the builds run on the same node,
  otherwise use `ReadWriteMany` (NFS, for example).
- An Ingress controller. cert-manager is optional, for a TLS certificate.
- A node with a few CPU cores for the builds. A build of `zfs-utils` and `zfs-linux` takes a few minutes.

## Steps

1. Create the namespace:

    ```bash
    kubectl create namespace arch-repo
    ```

2. Create the volume. Set `storageClassName` in `pvc.yaml` if the default StorageClass is not the one you want.

    ```bash
    kubectl apply -f deploy/k8s/pvc.yaml
    ```

3. Create the ServiceAccount and the Role that lets the server create Jobs, read pods and stream their logs:

    ```bash
    kubectl apply -f deploy/k8s/rbac.yaml
    ```

4. Check the environment variables in `deployment.yaml` (see [Configuration](../reference/configuration.md)), then
   start the server and its Service:

    ```bash
    kubectl apply -f deploy/k8s/deployment.yaml
    kubectl apply -f deploy/k8s/service.yaml
    ```

5. Replace `zfs.example.com` and the `ingressClassName` in `ingress.yaml`, then apply it:

    ```bash
    kubectl apply -f deploy/k8s/ingress.yaml
    ```

    The live build logs are Server-Sent Events. ingress-nginx and Traefik pass them through with their default
    settings; a proxy that buffers responses delivers a log only when the build has ended.

6. Open the domain in a browser and press **Check Now** on the dashboard. The server compares the kernels and starts
   a build for each kernel without packages. Click a running build to follow its log.

## Pin the builds to a node

Builds compile a kernel module and need CPU time; the server itself needs almost none. To run the builds on one node,
set `BUILD_NODE` to the node's `kubernetes.io/hostname`:

```yaml
- name: BUILD_NODE
  value: "build-node-1"
```

The Settings page of the web UI changes it at runtime, until the next restart.

## Updates

The image `ghcr.io/markusressel/arch-zfs-docker:latest` is published on every push to `main`. The Deployment uses
`imagePullPolicy: Always`, so a restart pulls the newest one:

```bash
kubectl rollout restart -n arch-repo deploy/arch-repo-server
```

Build Jobs copy the builder from the image of the server's pod, so they always run the same version as the server.

## Metrics

The server serves Prometheus metrics on port 9414. To scrape them, add the port to the Deployment and a Service and
ServiceMonitor for it, see [Monitoring and alerts](../guides/monitoring.md).
