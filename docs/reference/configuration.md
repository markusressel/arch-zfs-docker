# Configuration

The server reads its configuration from environment variables or from command line flags; a flag wins over the
variable. `arch-repo-server -h` lists the flags.

| Variable | Flag | Default | Meaning |
|---|---|---|---|
| `LISTEN_ADDR` | `-listen` | `:8080` | Address of the web UI, the API and the pacman repository. |
| `METRICS_ADDR` | `-metrics-listen` | `:9414` | Address of the Prometheus metrics (`/metrics`). Empty disables them. |
| `REPO_DIR` | `-repo-dir` | `/repo` | Directory of the repository, the caches, the build logs and `compat.json`. |
| `REPO_NAME` | `-repo-name` | `zfslocal` | Name of the pacman repository: the section in `pacman.conf` and the database file `zfslocal.db.tar.zst`. |
| `AUTO_CHECK_INTERVAL` | `-auto-check-interval` | `6h` | How often the server checks Arch Linux for new kernels. `0` disables the checks. |
| `BUILD_RUNNER` | `-build-runner` | `auto` | Where builds run: `kubernetes`, `docker`, or `auto` (Kubernetes when credentials are found, otherwise Docker). |
| `K8S_NAMESPACE` | `-namespace` | `arch-repo` | Namespace of the build Jobs. |
| `BUILD_NODE` | `-build-node` | | `kubernetes.io/hostname` of the node the build Jobs run on. Empty lets Kubernetes choose. |
| `BUILDER_IMAGE` | `-builder-image` | the image of the server's pod | Image the build Jobs copy the builder (`/app/arch-repo`) from. |
| `SIGNING_SECRET` | `-signing-secret` | | Secret (key `key.asc`) with the private key the build Jobs sign packages and database with, see [Signing the packages](../guides/signing.md). Empty: unsigned. |
| `KUBECONFIG` | `-kubeconfig` | `~/.kube/config` | kubeconfig for a server outside the cluster. Inside the cluster the ServiceAccount is used. |
| `K8S_DASHBOARD_URL` | `-dashboard-url` | | Base URL of a Kubernetes dashboard; the build list links each Job there. |
| `DOCKER_HOST` | `-docker-host` | `unix:///var/run/docker.sock` | Docker engine of the Docker runner. |
| `DOCKER_REPO_MOUNT` | `-docker-repo-mount` | `REPO_DIR` | Host path or named volume mounted at `/repo` in build containers. Needed when the server runs in a container itself. |
| `DEV_MODE` | `-dev` | `false` | Fake build jobs that build nothing, for working on the web UI. |

## Settings in the web UI

The Settings page changes the check interval and the build node while the server runs. The values last until the
server restarts; for good, set the variables above.

## The repository directory

```text
/repo
├── zfslocal/x86_64/   packages, their signatures (.sig), the pacman database (zfslocal.db.tar.zst, signed: .sig)
├── cache/pacman/      pacman packages, shared by all builds
├── cache/sources/     OpenZFS source tarballs
├── logs/              one log per build, <job>.log
└── compat.json        kernel and OpenZFS pairs that failed because of the kernel
```

The server serves `zfslocal/x86_64/` as the pacman repository. Packages are served with a cache lifetime of 30 days
(their names never change), the database with no caching.
