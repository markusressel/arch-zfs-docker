# AGENTS.md — arch-zfs-docker

Go server, Vue web UI and Kubernetes/Docker build runner that builds and serves Arch Linux `zfs-linux` and `zfs-utils`
packages. The documentation in `docs/` (Zensical, published to GitHub Pages) is the reference; read the page of the
area before changing it:

| Area | Page |
|---|---|
| Commands, checks before a commit, releases | `docs/development/index.md` |
| Packages, serving rules, Kubernetes, web UI conventions, OpenZFS support, package removal | `docs/development/architecture.md` |
| Build jobs: steps, runners and fakes, log lines the classifier needs, signatures, static binary, vercmp | `docs/development/builder.md` |
| Environment variables and flags, the repository directory | `docs/reference/configuration.md` |
| API, metrics | `docs/reference/api.md`, `docs/reference/metrics.md` |
| Failure categories, OpenZFS states, held-back updates | `docs/guides/` |

## Rules

- Before a commit: `go test ./...`, `gofmt -l .` (no output); with changes in `web/` also `just test-ui` and
  `just build-ui`; with changes in `docs/` also `just docs` (strict, fails on broken links).
- Update the docs with every change of behavior. A new page goes into the section of its kind and into `nav` in
  `zensical.toml`. Links to files outside `docs/` are absolute GitHub URLs.
- The builder's package installs must stay full upgrades (`pacman -Su`), never `-Sy` followed by `-S`.
- Keep the log lines `internal/compat` parses unchanged (`TestBuildLogIsUnderstoodByTheCompatClassifier`).
- Never pass `--skippgpcheck` to makepkg.
- No private host names, cluster IP addresses, local absolute paths or machine names in the repository; examples use
  `zfs.example.com`.
