# Monitoring and alerts

The server serves Prometheus metrics on a port of its own, 9414 by default (`METRICS_ADDR`). The Ingress of the web UI
does not expose them. [Metrics](../reference/metrics.md) lists every metric.

## Scrape the metrics in Kubernetes

Add the port to the container in `deployment.yaml` (the manifest in `deploy/k8s/` has it):

```yaml
ports:
- containerPort: 8080
  name: http
- containerPort: 9414
  name: metrics
```

A Service for the port, and a ServiceMonitor for the Prometheus Operator. The label `release: prometheus` is what
kube-prometheus-stack selects by default; use the label your Prometheus selects ServiceMonitors by.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: arch-repo-metrics
  namespace: arch-repo
  labels:
    app: arch-repo-metrics
spec:
  ports:
  - name: metrics
    port: 9414
    targetPort: metrics
  selector:
    app.kubernetes.io/name: arch-repo-server
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: arch-repo-metrics
  namespace: arch-repo
  labels:
    release: prometheus
spec:
  selector:
    matchLabels:
      app: arch-repo-metrics
  endpoints:
  - port: metrics
    interval: 60s
```

Without the Operator, scrape `http://<server>:9414/metrics` with a static target.

## Alerts

The rules below are written as Prometheus alerting rules. In Grafana alerting, use each `expr` as the query and the
part after the comparison as the threshold. Each expression aggregates `by (kernel_pkg)`, so a restarted server pod
(with a new `pod` label) does not end an alert and start it again.

```yaml
groups:
- name: arch-repo
  rules:
  # Two failed builds in a row whose newest failure is not a kernel incompatibility: a problem of the builder.
  # With one check every 6 hours, this fires within about 6 hours of the first failure.
  - alert: ZfsRepoBuildsFailing
    expr: >
      max by (kernel_pkg) (arch_repo_build_fail_streak)
      * on (kernel_pkg) sum by (kernel_pkg) (arch_repo_build_last_failed{category!="kernel"})
      > 1
    labels:
      severity: warning
    annotations:
      summary: "{{ $labels.kernel_pkg }}: {{ $value }} builds failed in a row"

  # No package for the newest kernel for a day, although OpenZFS is not known to refuse it.
  - alert: ZfsRepoBehindUpstream
    expr: >
      max by (kernel_pkg) (arch_repo_kernel_up_to_date) == 0
      unless on (kernel_pkg) max by (kernel_pkg) (arch_repo_kernel_compat{state="incompatible"}) == 1
    for: 24h
    labels:
      severity: warning

  # OpenZFS refused the newest kernel: updates wait for a new OpenZFS release.
  - alert: ZfsRepoKernelUnsupported
    expr: max by (kernel_pkg, zfs_version) (arch_repo_kernel_compat{state="incompatible"}) == 1
    labels:
      severity: info

  # The scheduler stopped checking (it checks every 6 hours by default).
  - alert: ZfsRepoCheckStale
    expr: time() - max by (kernel_pkg) (arch_repo_upstream_checked_timestamp_seconds) > 13 * 3600
    labels:
      severity: warning

  - alert: ZfsRepoMetricsMissing
    expr: absent(up{job="arch-repo-metrics"} == 1)
    for: 15m
    labels:
      severity: warning
```

## A message when full upgrades work again

While the repository has no `zfs-linux` for the newest kernel, clients have to update with `--ignore` (see
[Updating while ZFS is held back](held-back-updates.md)). An alert on that state, sent with its resolved message,
tells you both when to start and when to stop:

```yaml
  - alert: ZfsRepoUpdatesOnHold
    expr: 1 - max by (kernel_pkg) (arch_repo_kernel_up_to_date{kernel_pkg="linux"}) > 0
    for: 30m
    labels:
      severity: info
    annotations:
      summary: >
        No zfs-linux for the newest Arch kernel yet. Update with:
        sudo pacman -Syu --ignore linux,linux-headers,zfs-linux,zfs-linux-headers,zfs-utils
```

Alertmanager sends the resolved message when `send_resolved` is set on the receiver; that is the moment a plain
`sudo pacman -Syu` works again. `for: 30m` skips the few minutes of a normal build after each kernel release. Select
the `kernel_pkg` your machines run.

Grafana sends resolved messages too, with the annotations of the firing alert. To word the resolved message
differently, add an annotation such as `resolved_summary` and pick it in the notification template:

```text
{{ if and (eq .Status "resolved") .Annotations.resolved_summary }}{{ .Annotations.resolved_summary }}{{ else }}{{ .Annotations.summary }}{{ end }}
```
