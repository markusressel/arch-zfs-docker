# Development

The server is written in Go, the web UI in Vue 3 with TypeScript, Vite and the component library
[Vuetify](https://vuetifyjs.com). The UI is built into `internal/ui/dist` and embedded in the Go binary. [just](https://just.systems) runs the common tasks.

## Commands

| Task | Command |
|---|---|
| Install the UI dependencies | `just install-ui` |
| Build the UI (needed once before the server serves a UI) | `just build-ui` |
| Build the server binary with the UI | `just build` |
| Server with real builds in the local Docker engine, UI with hot reload on http://localhost:5173/ui/ | `just dev` |
| Server with the embedded UI, like production | `just dev-embedded` |
| Server with fake build jobs, nothing is built | `just dev-mock` |
| Go tests | `go test ./...` or `just test` (with the race detector) |
| UI type check and tests | `just test-ui` |
| Format Go code | `just fmt` |
| Take the screenshots of the README and these docs, in the dark theme (needs a running UI, Firefox and ImageMagick) | `just screenshots` |
| Build these docs, as CI does | `just docs` |
| Preview these docs on http://localhost:8001 | `just docs-serve` |

`just dev` keeps the repository in `.dev/repo`. The UI on `:8080` is the output of the last `just build-ui`; the one on
`:5173` is live and sends `/api` to the server.

## Before a commit

- `go test ./...` and `gofmt -l .` (no output).
- With changes in `web/`: `just test-ui` and `just build-ui`.
- With changes to the docs: `just docs`, which fails on broken links.
- Update the docs with every change of behavior: these pages, the README and `deploy/k8s/`.
- No private host names, IP addresses, local paths or machine names in the repository. Examples use
  `zfs.example.com`.

## Releases

There are no versions. Every push to `main` publishes the image `ghcr.io/markusressel/arch-zfs-docker:latest`
(`.github/workflows/publish-service.yml`) and these docs (`.github/workflows/docs.yml`). Pull requests run the tests
(`.github/workflows/test.yml`).

## More

- [Architecture](architecture.md): the parts of the server and the rules they follow.
- [The builder](builder.md): what a build job does, and how it is tested.
