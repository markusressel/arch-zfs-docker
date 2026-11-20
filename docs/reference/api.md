# HTTP API

The web UI uses this JSON API, and scripts can use it too. It has no authentication: anyone who reaches the server
can start builds and delete packages. Keep the server in a network you trust, or put authentication in front of it,
for example at the Ingress.

## Repository

| Request | Response |
|---|---|
| `GET /api/packages` | The repository: `packageCount`, `totalSizeBytes`, `dbLastModified` and `packages`, each with file name, package name, versions, size and download URL. |
| `DELETE /api/packages/{filename}` | Starts a job that deletes the file and rebuilds the pacman database. Afterwards the server checks upstream again, so a deleted package for the newest kernel is built again. |
| `GET /{repo}/x86_64/...` | The pacman repository itself: packages and the database. |

## Kernels and builds

| Request | Response |
|---|---|
| `GET /api/upstream` | Per kernel package (`linux`, `linux-lts`): `upstreamKernel`, `localKernel`, `upToDate`, `checkedAt` and, while not up to date, `compat` (see [New kernels and OpenZFS support](../guides/new-kernels.md)). |
| `POST /api/upstream/check` | Checks Arch Linux now and starts the builds that are needed. Returns `{"status": "check_started"}`. |
| `GET /api/builds` | All build and maintenance jobs, newest first: `name`, `status` (`Pending`, `Running`, `Succeeded`, `Failed`), `startTime`, `endTime`, `durationSec`, `variant`, and for failed builds `failure` with `category`, `reason` and `evidence` (see [Failed builds](../guides/failed-builds.md)). |
| `POST /api/builds` | Starts a build, see below. `202` with the job. |
| `DELETE /api/builds/{name}` | Deletes a finished job. |
| `DELETE /api/builds` | Deletes all finished jobs. |
| `GET /api/builds/{name}/logs` | The log of a job as Server-Sent Events, live while it runs. |
| `GET /api/kernels?variant=lts` | Kernel versions in the Arch Linux Archive, for the build form. |
| `GET /api/zfs-versions` | OpenZFS releases, for the build form. |

### Start a build

```bash
curl -X POST https://zfs.example.com/api/builds \
  -H 'Content-Type: application/json' \
  -d '{"variant": "lts", "kernelVersion": "", "zfsVersion": ""}'
```

| Field | Meaning |
|---|---|
| `variant` | `""` for `linux`, `"lts"` for `linux-lts`. |
| `kernelVersion` | A kernel version from the Arch Linux Archive, or `""` for the newest. |
| `zfsVersion` | An OpenZFS release such as `2.4.4`, or `""` for the version of the AUR package. |
| `forceBuild` | Build even if the repository has the package already. Without it the server answers `409` with `"error": "already_built"`. |
| `forceRebuildUtils` | Build `zfs-utils` again, too. |
| `allowUnsupportedKernel` | Skip OpenZFS' kernel version check (`--enable-linux-experimental`). |

## Settings and health

| Request | Response |
|---|---|
| `GET /api/settings` | The settings of the server, see [Configuration](configuration.md), and `signing`: `enabled` when `SIGNING_SECRET` is set, plus `keyPath` and `fingerprint` of the public key once a signed build has published it. |
| `POST /api/settings` | Changes `autoCheckInterval` and `buildNode` until the next restart; answers like `GET`. |
| `GET /healthz` | `OK`, for probes. |
| `GET /metrics` on `METRICS_ADDR` | Prometheus metrics, see [Metrics](metrics.md). |
