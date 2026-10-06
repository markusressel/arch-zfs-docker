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
}

export interface BuildRequest {
  kernelVersion: string
  /** Empty for the AUR default */
  zfsVersion: string
  variant: string
  forceBuild: boolean
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

export interface VariantStatus {
  variant: string
  kernelPkg: string
  upstreamKernel: string
  localKernel: string
  upToDate: boolean
  checkedAt: string
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
}

export type SettingsUpdate = Pick<Settings, 'autoCheckInterval' | 'buildNode'>

/** Body of the 409 response when the requested kernel was already built. */
export interface AlreadyBuilt {
  error: 'already_built'
  kernelVersion: string
  package: PackageInfo
}
