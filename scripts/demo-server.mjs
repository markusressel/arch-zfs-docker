#!/usr/bin/env node
// A demo of the web UI: serves the built UI (internal/ui/dist, `just build-ui`) and an API with made-up data of a
// healthy repository. Both kernels are up to date, and a month of builds has one failure that the next check
// fixed. The screenshots of the README and the docs are taken from it (scripts/screenshots.mjs), so they do not
// depend on the state of a real instance.
//
// Usage: node scripts/demo-server.mjs [port]   (default 8099), then open http://localhost:8099/ui/
// Write requests (start a build, delete, save settings) answer 403: nothing in the demo changes.

import fs from 'node:fs'
import http from 'node:http'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const DIST = path.join(path.dirname(fileURLToPath(import.meta.url)), '..', 'internal', 'ui', 'dist')
const HOUR = 3600 * 1000
const DAY = 24 * HOUR
const ZFS = '2.4.4'
const NAMESPACE = 'arch-repo'
const REPO = 'zfslocal'

// Deterministic "random" numbers, so every run produces the same job names and durations.
function rng(seed) {
  let s = seed
  return () => {
    s = (s * 1103515245 + 12345) % 2147483648
    return s / 2147483648
  }
}

function humanSize(bytes) {
  return bytes >= 1024 * 1024 ? `${(bytes / 1024 / 1024).toFixed(2)} MiB` : `${(bytes / 1024).toFixed(2)} KiB`
}

/** Kernel releases of the last weeks: [days ago, upstream version] per variant, oldest first. */
const RELEASES = {
  '': [
    [33, '7.2.8.arch1-1'],
    [27, '7.2.9.arch1-1'],
    [22, '7.2.10.arch1-1'],
    [16, '7.2.11.arch1-1'],
    [11, '7.2.12.arch1-1'],
    [6, '7.2.13.arch1-1'],
    [1.5, '7.2.14.arch1-1'],
  ],
  lts: [
    [31, '6.18.58-1'],
    [25, '6.18.59-1'],
    [19, '6.18.60-1'],
    [13, '6.18.61-1'],
    [9, '6.18.62-1'],
    [4, '6.18.63-1'],
    [0.6, '6.18.64-1'],
  ],
}

/** Builds the API state relative to `now`, so the dates in the UI are always recent. */
export function demoState(now = Date.now()) {
  const random = rng(42)
  // The scheduler checks every 6 hours; the last check was 40 minutes ago.
  const lastCheck = Math.floor((now - 40 * 60 * 1000) / 60000) * 60000
  const nextCheckAfter = (t) => lastCheck - Math.floor((lastCheck - t) / (6 * HOUR)) * 6 * HOUR
  const jobName = (t) => `zfs-build-${Math.floor(t / 1000)}-${Math.floor(random() * 0xffffff).toString(16).padStart(6, '0')}`

  const builds = []
  const packages = []
  const upstream = {}
  for (const [variant, releases] of Object.entries(RELEASES)) {
    const kernelPkg = variant ? `linux-${variant}` : 'linux'
    releases.forEach(([daysAgo, version], i) => {
      let start = nextCheckAfter(now - daysAgo * DAY)
      // One build failed while fetching packages (a mirror was down); the next check built it.
      if (variant === '' && i === 2) {
        const duration = 30 + Math.floor(random() * 10)
        builds.push({
          name: jobName(start),
          namespace: NAMESPACE,
          status: 'Failed',
          startTime: new Date(start).toISOString(),
          endTime: new Date(start + duration * 1000).toISOString(),
          durationSec: duration,
          variant,
          failure: {
            category: 'environment',
            reason: 'Installing the build dependencies failed (pacman, mirror or network)',
            evidence: `error: failed retrieving file 'linux-headers-${version}-x86_64.pkg.tar.zst' from mirror : Connection timed out`,
          },
        })
        start += 6 * HOUR
      }
      const duration = 190 + Math.floor(random() * 80)
      const end = start + duration * 1000
      builds.push({
        name: jobName(start),
        namespace: NAMESPACE,
        status: 'Succeeded',
        startTime: new Date(start).toISOString(),
        endTime: new Date(end).toISOString(),
        durationSec: duration,
        variant,
      })
      // The repository keeps the packages of the last four kernels.
      if (i >= releases.length - 4) {
        const pkgver = version.replace('-', '.')
        for (const [name, size] of [
          [`zfs-${kernelPkg}-headers`, 463000 + Math.floor(random() * 2000)],
          [`zfs-${kernelPkg}`, 2650000 + Math.floor(random() * 20000)],
        ]) {
          const filename = `${name}-${ZFS}_${pkgver}-1-x86_64.pkg.tar.zst`
          packages.push({
            filename,
            packageName: name,
            version: ZFS,
            kernelVersion: pkgver,
            arch: 'x86_64',
            sizeBytes: size,
            sizeHuman: humanSize(size),
            modTime: new Date(end).toISOString(),
            downloadUrl: `/${REPO}/x86_64/${filename}`,
          })
        }
      }
    })
    const latest = releases.at(-1)[1]
    upstream[kernelPkg] = {
      variant,
      kernelPkg,
      upstreamKernel: latest,
      localKernel: latest.replace('-', '.'),
      upToDate: true,
      checkedAt: new Date(lastCheck).toISOString(),
    }
  }

  const utilsTime = new Date(now - 34 * DAY).toISOString()
  for (const [name, size] of [
    ['zfs-utils', 3337843],
    ['zfs-utils-debug', 9876543],
  ]) {
    const filename = `${name}-${ZFS}-1-x86_64.pkg.tar.zst`
    packages.push({
      filename,
      packageName: name,
      version: ZFS,
      arch: 'x86_64',
      sizeBytes: size,
      sizeHuman: humanSize(size),
      modTime: utilsTime,
      downloadUrl: `/${REPO}/x86_64/${filename}`,
    })
  }

  builds.sort((a, b) => b.startTime.localeCompare(a.startTime))
  packages.sort((a, b) => b.modTime.localeCompare(a.modTime) || a.filename.localeCompare(b.filename))
  const total = packages.reduce((sum, p) => sum + p.sizeBytes, 0)

  return {
    '/api/settings': {
      autoCheckInterval: '6h',
      buildNode: '',
      repoName: REPO,
      namespace: NAMESPACE,
      listenAddr: ':8080',
      repoDir: '/repo',
      runner: 'kubernetes',
    },
    '/api/upstream': upstream,
    '/api/builds': builds,
    '/api/packages': {
      repoName: REPO,
      arch: 'x86_64',
      packageCount: packages.length,
      totalSizeBytes: total,
      totalSizeHuman: humanSize(total),
      dbLastModified: packages[0].modTime,
      packages,
    },
    '/api/kernels': { versions: RELEASES[''].map(([, v]) => v).reverse() },
    '/api/kernels?variant=lts': { versions: RELEASES.lts.map(([, v]) => v).reverse() },
    '/api/zfs-versions': { versions: [ZFS, '2.4.3', '2.3.9'] },
  }
}

const TYPES = { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css', '.svg': 'image/svg+xml', '.woff2': 'font/woff2' }

function serveUi(url, res) {
  const rel = url.pathname.replace(/^\/ui\/?/, '') || 'index.html'
  const file = path.join(DIST, path.normalize(rel))
  if (!file.startsWith(DIST) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) {
    res.writeHead(404).end()
    return
  }
  res.writeHead(200, { 'content-type': TYPES[path.extname(file)] ?? 'application/octet-stream' })
  fs.createReadStream(file).pipe(res)
}

function serveLog(name, res) {
  res.writeHead(200, { 'content-type': 'text/event-stream', 'cache-control': 'no-store' })
  for (const line of [`==> ${name}: demo build log`, '==> Building zfs-linux', '==> Finished making: zfs-linux']) res.write(`data: ${line}\n\n`)
  res.end('event: close\ndata: \n\n')
}

/** Starts the demo server; resolves to the http.Server once it listens. Port 0 picks a free port. */
export async function startDemoServer(port = 8099, host = '127.0.0.1') {
  if (!fs.existsSync(path.join(DIST, 'index.html'))) {
    throw new Error(`no built UI in ${path.relative(process.cwd(), DIST)}; run \`just build-ui\` first`)
  }
  const server = http.createServer((req, res) => {
    const url = new URL(req.url, 'http://demo')
    if (url.pathname === '/' || url.pathname === '/ui') {
      res.writeHead(302, { location: '/ui/' }).end()
    } else if (url.pathname.startsWith('/ui/')) {
      serveUi(url, res)
    } else if (req.method !== 'GET') {
      res.writeHead(403, { 'content-type': 'application/json' }).end(JSON.stringify({ error: 'read-only demo' }))
    } else if (/^\/api\/builds\/[^/]+\/logs$/.test(url.pathname)) {
      serveLog(decodeURIComponent(url.pathname.split('/')[3]), res)
    } else {
      const key = url.pathname === '/api/kernels' && url.searchParams.get('variant') ? `${url.pathname}?variant=lts` : url.pathname
      const body = demoState()[key]
      if (body === undefined) res.writeHead(404).end()
      else res.writeHead(200, { 'content-type': 'application/json' }).end(JSON.stringify(body))
    }
  })
  await new Promise((resolve, reject) => {
    server.once('error', reject)
    server.listen(port, host, resolve)
  })
  return server
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const server = await startDemoServer(Number(process.argv[2] ?? 8099)).catch((err) => {
    console.error(`error: ${err.message}`)
    process.exit(1)
  })
  console.log(`Demo UI on http://localhost:${server.address().port}/ui/`)
}
