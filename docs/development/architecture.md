# Architecture

One Go binary has two modes. Without arguments it is the server: web UI, API, pacman repository, scheduler and
metrics. With `builder ...` it is the builder that runs inside a build job (`cmd/server/main.go` decides). Build jobs
copy the binary from the server's image, so server and builder always have the same version.

## Packages

```text
cmd/server/main.go       entry point, switches to builder mode for `builder ...`
internal/
  builder/               the build job: kernel, build user, makepkg, publishing (see The builder)
  compat/                OpenZFS support: META, AUR and GitHub sources, the log classifier, compat.json
  config/                environment variables and flags
  k8s/                   runners: Kubernetes Jobs, Docker containers, a mock; log streaming
  kernel/                kernel version normalization, versions of the Arch Linux Archive
  repo/                  scans the repository directory: packages, sizes, the database
  scheduler/             checks archlinux.org, compares with the repository, starts builds
  server/                HTTP: API, Server-Sent Events, pacman files, metrics (metrics.go)
  ui/                    embeds the built web UI (dist/ is build output, not in git)
  zfs/                   OpenZFS release versions
web/src/
  demo/                  made-up data for the demo build (`vite build --mode demo`), which runs without a server
  plugins/vuetify.ts     the Vuetify themes (GitHub's light and dark palettes) and component defaults
  styles/                the reset (lowest CSS layer), CSS variables mapped onto the theme, shared styles
  api/                   typed fetch wrappers, nothing else
  types/                 the JSON of the API
  composables/           shared state and behavior (useBuilds, useLogStream, useStickToBottom, ...)
  utils/                 pure, unit-tested logic (formatting, estimates, analytics, pacman commands)
  components/            one folder per area: ui, layout, dashboard, install, settings, analytics, logs, dialogs
  views/                 one view per route: Dashboard, Install, Builds, Analytics, Settings
deploy/k8s/              the Kubernetes manifests
builder/                 the shell scripts of the first version (not maintained with the server)
docs/                    this site (Zensical, zensical.toml)
```

## Rules

### Serving the repository

- Package files (`*.pkg.tar*`) are served with `Cache-Control: public, max-age=2592000, immutable`: a file name is
  never reused for other content.
- The database files (`*.db*`, `*.files*`) are served with `no-store`, so pacman always sees the current state.

### Web UI

- Vuetify puts its styles into CSS cascade layers. `main.ts` imports the Vuetify plugin before `App.vue`, so
  `vuetify/styles` declares the layer order first; otherwise the reset layer beats the component styles.
- Unlayered CSS always beats Vuetify. Component styles must not style bare elements (`button`, `input`, `*`) that
  Vuetify components contain. The global reset lives in the layer `app-reset`, declared before Vuetify's layers.
- Typography classes are the Material 3 ones of Vuetify 4 (`text-body-small`, `text-title-medium`); the old
  `text-caption` and `text-h6` no longer exist.
- The demo build answers the API in the browser (`api/http.ts`, `demo/data.ts`). Its branches test
  `import.meta.env.MODE === 'demo'`, which Vite replaces at build time, so the normal build leaves them out.
- Colors come from the theme: `rgb(var(--v-theme-success))` and so on. `styles/variables.css` maps the older
  `--status-*`, `--text-*` and `--bg-*` variables onto it.

### Kubernetes

- Inside the cluster the server uses its ServiceAccount and plain REST calls (`/apis/batch/v1/...`), no client
  library.
- Build logs come from `/api/v1/namespaces/{ns}/pods/{pod}/log?follow=true` and go to the browser as Server-Sent
  Events. When the browser disconnects, the request to Kubernetes is cancelled through the context.
- A failed Job has no `completionTime`; its end is the time of its `Failed` condition.

### The web UI

- Build logs can be fast. The log view collects chunks per animation frame (`requestAnimationFrame`) and keeps a
  bounded scrollback (`useLogStream`, `utils/logBuffer.ts`).
- The log view scrolls with new output only while the user is at the bottom (`useStickToBottom`).
- One component per concern in the matching `components/` folder, shared state in `composables/`, logic as pure
  functions in `utils/` with a `*.test.ts` next to them.

### OpenZFS support

- Predicted from `Linux-Maximum` in the `META` file of the OpenZFS release that the AUR packages. A prediction only
  warns; the build runs and OpenZFS' `configure` decides.
- Learned from failed builds: `compat` classifies each log. A failure caused by the kernel stores the pair of kernel
  and OpenZFS version in `/repo/compat.json`, and the scheduler does not build that pair again until a new OpenZFS
  release, another kernel, or 7 days later.
- `--enable-linux-experimental` is used only when a request sets `allowUnsupportedKernel`; the scheduler never does.

### Package removal

`DELETE /api/packages/{filename}` starts a maintenance job with the same runner as the builds, so it has the file
permissions of the repository and the pacman tools. It deletes the file and rebuilds the database; then the upstream
check runs and rebuilds the package if it was the one for the newest kernel. Maintenance jobs never count as builds:
not in estimates, statistics, metrics or the scheduler.

### Upstream checks

The scheduler asks `https://archlinux.org/packages/core/x86_64/<pkg>/json/` and starts no container and no pacman for
that. It starts a build only when the newest kernel has no package in the repository.
