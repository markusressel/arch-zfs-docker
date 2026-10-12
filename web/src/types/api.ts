// Mirrors the JSON shapes served by the Go backend (internal/server, internal/k8s, internal/repo, internal/scheduler).

export type JobStatus = 'Pending' | 'Running' | 'Succeeded' | 'Failed'

export interface JobSummary {
  name: string
  namespace: string
  status: JobStatus
  startTime?: string
  endTime?: string
  durationSec: number
  podName?: string
  /** "" for standard linux, "lts" for linux-lts */
  variant: string
  /** Requested kernel version, empty for latest */
  kernelVersion?: string
  /** Requested OpenZFS version, empty for the AUR default */
  zfsVersion?: string
  /** Empty for builds; e.g. "remove-package" for repository maintenance jobs */
  task?: string
  /** Package file a maintenance task works on */
  package?: string
  /** The build may target a kernel newer than the OpenZFS release supports */
  unsupportedKernel?: boolean
}

export interface BuildRequest {
  kernelVersion: string
  /** Empty for the AUR default */
  zfsVersion: string
  variant: string
  forceBuild: boolean
  /** Skip OpenZFS' check for the newest supported kernel (--enable-linux-experimental) */
  allowUnsupportedKernel?: boolean
}

export interface PackageInfo {
  filename: string
  packageName: string
  version: string
  kernelVersion?: string
  arch: string
  sizeBytes: number
  sizeHuman: string
  modTime: string
  sha256?: string
  downloadUrl: string
}

export interface RepoSummary {
  repoName: string
  arch: string
  packageCount: number
  totalSizeBytes: number
  totalSizeHuman: string
  dbLastModified?: string
  packages: PackageInfo[]
}

export type CompatState = 'ok' | 'predicted_unsupported' | 'incompatible' | 'unknown'

export interface CompatIssue {
  number: number
  title: string
  url: string
  state: 'open' | 'closed' | string
}

/** Whether the packaged OpenZFS release supports the upstream kernel (only set while a build is needed). */
export interface CompatInfo {
  state: CompatState
  zfsVersion?: string
  /** Linux-Maximum (major.minor) declared by the ZFS release */
  linuxMax?: string
  reason?: string
  /** Failed build job the verdict was learned from */
  job?: string
  since?: string
  issues?: CompatIssue[]
  issuesUrl?: string
}

export interface VariantStatus {
  variant: string
  kernelPkg: string
  upstreamKernel: string
  localKernel: string
  upToDate: boolean
  checkedAt: string
  compat?: CompatInfo
}

/** Keyed by kernel package name, e.g. "linux", "linux-lts". */
export type UpstreamStatus = Record<string, VariantStatus>

export interface Settings {
  autoCheckInterval: string
  buildNode: string
  repoName: string
  namespace: string
  listenAddr: string
  repoDir: string
  dashboardUrl?: string
  /** Where builds run: Kubernetes, the local Docker engine, or the dev mock. */
  runner?: 'kubernetes' | 'docker' | 'mock'
}

export type SettingsUpdate = Pick<Settings, 'autoCheckInterval' | 'buildNode'>

/** Body of the 409 response when the requested kernel was already built. */
export interface AlreadyBuilt {
  error: 'already_built'
  kernelVersion: string
  package: PackageInfo
}
