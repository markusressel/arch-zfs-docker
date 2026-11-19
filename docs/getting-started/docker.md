# Run with Docker

Without Kubernetes, the server runs each build as a container in a Docker engine. This suits a single machine, or
trying the project out before a cluster deployment.

## The server as a binary

Build the server with the web UI, then start it with a directory for the repository:

```bash
just install-ui
just build
bin/arch-repo-server -repo-dir ~/zfs-repo -listen :8080
```

The web UI is on http://localhost:8080/ui/ and the repository on http://localhost:8080/zfslocal/x86_64/.

With `BUILD_RUNNER=auto` (the default) the server uses Kubernetes when it finds credentials and Docker otherwise.
`-build-runner docker` forces Docker, so a kubeconfig on the machine is never used.

## How a build container works

Each build is a container of `archlinux:base-devel`, like a Kubernetes Job. The server uploads its own executable into
the container and runs it in builder mode (`<binary> builder build`). That only works with a statically linked
binary: `just build` and the Dockerfile set `CGO_ENABLED=0`, a plain `go build` may not.

The repository directory is bind-mounted at `/repo` in every build container, so the path has to exist on the Docker
host.

## The server in a container

When the server itself runs in a container, give it the Docker socket and tell it which host path is behind its
`/repo`, because the build containers are started by the host's Docker engine:

```bash
docker run -d --name arch-repo \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /srv/zfs-repo:/repo \
  -e BUILD_RUNNER=docker \
  -e DOCKER_REPO_MOUNT=/srv/zfs-repo \
  ghcr.io/markusressel/arch-zfs-docker:latest
```

!!! warning
    Access to the Docker socket is root access to the host. Run this only on a machine where that is acceptable.

## The scripts in `builder/`

[`builder/`](https://github.com/markusressel/arch-zfs-docker/tree/main/builder) holds the shell scripts of the first
version, which build the packages into a local directory without the server:

```bash
cd builder
REPOSITORY_NAME="zfslocal" REPOSITORY_PATH=~/zfs-repo ./populate-package-repository.sh
```

They are not maintained together with the server and know nothing about OpenZFS support or failed builds.
