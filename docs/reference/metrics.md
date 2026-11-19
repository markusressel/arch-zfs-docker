# Metrics

`/metrics` on `METRICS_ADDR` (`:9414` by default) in the Prometheus text format. The server computes the values at
every scrape from the same data as the web UI. All metrics are gauges.

## Kernels

One series per kernel package, label `kernel_pkg` (`linux`, `linux-lts`). They appear after the first upstream check.

| Metric | Labels | Value |
|---|---|---|
| `arch_repo_kernel_up_to_date` | `kernel_pkg` | `1` when the repository has packages for the newest kernel of Arch Linux. |
| `arch_repo_kernel_info` | `kernel_pkg`, `upstream`, `local` | Always `1`; the labels hold the newest kernel of Arch Linux and the newest one with packages here. |
| `arch_repo_kernel_compat` | `kernel_pkg`, `state`, `zfs_version` | `1` for the current state, `0` for the other three (`ok`, `predicted_unsupported`, `incompatible`, `unknown`). A kernel that is up to date is `ok`. |
| `arch_repo_upstream_checked_timestamp_seconds` | `kernel_pkg` | Time of the last check against Arch Linux. |

## Builds

Counted from the build jobs that still exist; maintenance jobs (package removal) are left out. Deleting finished jobs
in the web UI changes these values.

| Metric | Labels | Value |
|---|---|---|
| `arch_repo_build_jobs` | `kernel_pkg`, `status` | Jobs per status (`Pending`, `Running`, `Succeeded`, `Failed`). |
| `arch_repo_build_fail_streak` | `kernel_pkg` | Failed builds since the last successful one. |
| `arch_repo_build_last_failed` | `kernel_pkg`, `category` | `1` for the cause of the newest finished build when it failed (`environment`, `kernel`, `build`, or `unknown` when the log could not be read), `0` for the others. All `0` when the newest build succeeded. |
| `arch_repo_build_last_success_timestamp_seconds` | `kernel_pkg` | End of the newest successful build, `0` if there is none. |
| `arch_repo_build_last_finished_timestamp_seconds` | `kernel_pkg` | End of the newest finished build. |
| `arch_repo_build_last_duration_seconds` | `kernel_pkg` | Duration of the newest finished build. |
| `arch_repo_jobs_list_ok` | | `1` when the jobs could be listed. When `0`, the build metrics are missing from the scrape. |

## Repository

| Metric | Value |
|---|---|
| `arch_repo_packages` | Package files in the repository. |
| `arch_repo_size_bytes` | Their total size. |
| `arch_repo_db_last_modified_timestamp_seconds` | Last change of the pacman database. |
| `arch_repo_index_ok` | `1` when the repository directory could be read. |
