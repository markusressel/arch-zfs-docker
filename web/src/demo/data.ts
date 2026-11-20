// Made-up data of a healthy repository for the demo build (`npm run build:demo`), which runs without a server: on
// GitHub Pages and for the screenshots. Both kernels are up to date, and a month of builds has one failure that
// the next check fixed. All dates are relative to the moment the data is made, so the demo always looks current.

import type { JobSummary, PackageInfo, RepoSummary, Settings, UpstreamStatus } from '../types/api'

const HOUR = 3600 * 1000
const DAY = 24 * HOUR
const ZFS = '2.4.4'
const NAMESPACE = 'arch-repo'
const REPO = 'zfslocal'

/** The host the pacman.conf snippet of the demo shows, as in the docs. */
export const DEMO_ORIGIN = 'https://zfs.example.com'

/** Kernel releases of the last weeks: [days ago, upstream version] per variant, oldest first. */
const RELEASES: Record<string, [number, string][]> = {
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

/** Deterministic "random" numbers, so job names and durations are the same on every load. */
function rng(seed: number) {
  let s = seed
  return () => {
    s = (s * 1103515245 + 12345) % 2147483648
    return s / 2147483648
  }
}

function humanSize(bytes: number) {
  return bytes >= 1024 * 1024 ? `${(bytes / 1024 / 1024).toFixed(2)} MiB` : `${(bytes / 1024).toFixed(2)} KiB`
}

function pkg(name: string, filename: string, size: number, modTime: number, kernelVersion?: string): PackageInfo {
  return {
    filename,
    packageName: name,
    version: ZFS,
    kernelVersion,
    arch: 'x86_64',
    sizeBytes: size,
    sizeHuman: humanSize(size),
    modTime: new Date(modTime).toISOString(),
    downloadUrl: `/${REPO}/x86_64/${filename}`,
  }
}

/** The answers of the API's GET endpoints, keyed by path (and query, for the kernel versions). */
export function demoResponses(now = Date.now()): Record<string, unknown> {
  const random = rng(42)
  // The scheduler checks every 6 hours; the last check was 40 minutes ago.
  const lastCheck = Math.floor((now - 40 * 60 * 1000) / 60000) * 60000
  const nextCheckAfter = (t: number) => lastCheck - Math.floor((lastCheck - t) / (6 * HOUR)) * 6 * HOUR
  const jobName = (t: number) =>
    `zfs-build-${Math.floor(t / 1000)}-${Math.floor(random() * 0xffffff)
      .toString(16)
      .padStart(6, '0')}`

  const builds: JobSummary[] = []
  const packages: PackageInfo[] = []
  const upstream: UpstreamStatus = {}
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
        const headers = `zfs-${kernelPkg}-headers`
        const module = `zfs-${kernelPkg}`
        packages.push(pkg(headers, `${headers}-${ZFS}_${pkgver}-1-x86_64.pkg.tar.zst`, 463000 + Math.floor(random() * 2000), end, pkgver))
        packages.push(pkg(module, `${module}-${ZFS}_${pkgver}-1-x86_64.pkg.tar.zst`, 2650000 + Math.floor(random() * 20000), end, pkgver))
      }
    })
    const latest = releases[releases.length - 1]![1]
    upstream[kernelPkg] = {
      variant,
      kernelPkg,
      upstreamKernel: latest,
      localKernel: latest.replace('-', '.'),
      upToDate: true,
      checkedAt: new Date(lastCheck).toISOString(),
    }
  }

  const utilsTime = now - 34 * DAY
  packages.push(pkg('zfs-utils', `zfs-utils-${ZFS}-1-x86_64.pkg.tar.zst`, 3337843, utilsTime))
  packages.push(pkg('zfs-utils-debug', `zfs-utils-debug-${ZFS}-1-x86_64.pkg.tar.zst`, 9876543, utilsTime))

  builds.sort((a, b) => b.startTime!.localeCompare(a.startTime!))
  packages.sort((a, b) => b.modTime.localeCompare(a.modTime) || a.filename.localeCompare(b.filename))
  const total = packages.reduce((sum, p) => sum + p.sizeBytes, 0)

  const settings: Settings = {
    autoCheckInterval: '6h',
    buildNode: '',
    repoName: REPO,
    namespace: NAMESPACE,
    listenAddr: ':8080',
    repoDir: '/repo',
    runner: 'kubernetes',
  }
  const repo: RepoSummary = {
    repoName: REPO,
    arch: 'x86_64',
    packageCount: packages.length,
    totalSizeBytes: total,
    totalSizeHuman: humanSize(total),
    dbLastModified: packages[0]!.modTime,
    packages,
  }
  return {
    '/api/settings': settings,
    '/api/upstream': upstream,
    '/api/builds': builds,
    '/api/packages': repo,
    '/api/kernels?variant=': { versions: RELEASES['']!.map(([, v]) => v).reverse() },
    '/api/kernels?variant=lts': { versions: RELEASES.lts!.map(([, v]) => v).reverse() },
    '/api/zfs-versions': { versions: [ZFS, '2.4.3', '2.3.9'] },
  }
}

/** The log a build job of the demo shows. */
export function demoLog(jobName: string): string[] {
  const failed = (demoResponses()['/api/builds'] as JobSummary[]).find((j) => j.name === jobName)?.failure
  return [
    `This is a demo with made-up data: ${jobName} never ran.`,
    '==> Installing the kernel and its headers',
    ...(failed
      ? [failed.evidence ?? failed.reason, 'error: failed to commit transaction']
      : ['==> Building zfs-utils', '==> Building zfs-linux', '==> Publishing the packages']),
  ]
}

/** Answers a request of the UI like the server would; requests that change something fail. */
export async function demoFetch(path: string, init?: RequestInit): Promise<Response> {
  if (init?.method && init.method !== 'GET') {
    return new Response('This is a demo with made-up data: nothing can be changed.', { status: 403 })
  }
  const body = demoResponses()[path]
  if (body === undefined) return new Response('not found', { status: 404 })
  return new Response(JSON.stringify(body), { headers: { 'content-type': 'application/json' } })
}
