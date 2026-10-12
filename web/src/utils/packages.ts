import type { JobSummary, PackageInfo } from '../types/api'

export function filterPackages(packages: PackageInfo[], term: string): PackageInfo[] {
  const q = term.toLowerCase().trim()
  if (!q) return packages
  return packages.filter(
    (p) =>
      p.packageName.toLowerCase().includes(q) ||
      p.version.toLowerCase().includes(q) ||
      (p.kernelVersion ?? '').toLowerCase().includes(q),
  )
}

/** Kernel version as it appears in package file names (all separators are dots). */
const dotted = (v: string) => v.trim().replace(/-/g, '.')

/** "lts" / "" for kernel module packages, null for kernel independent ones (zfs-utils). */
function packageVariant(packageName: string): string | null {
  if (packageName === 'zfs-linux' || packageName === 'zfs-linux-headers') return ''
  if (packageName === 'zfs-linux-lts' || packageName === 'zfs-linux-lts-headers') return 'lts'
  return null
}

/**
 * Whether a build job produced (or at least explicitly targeted) the package.
 * Jobs that asked for "latest" carry no kernel version and cannot be matched.
 */
export function jobBuiltPackage(job: JobSummary, pkg: PackageInfo): boolean {
  if (job.status !== 'Succeeded' || !job.kernelVersion || !pkg.kernelVersion) return false
  if (packageVariant(pkg.packageName) !== (job.variant || '')) return false
  if (job.zfsVersion && job.zfsVersion !== pkg.version) return false
  return dotted(job.kernelVersion) === dotted(pkg.kernelVersion)
}

/** Newest succeeded job (list is newest first) that built the package. */
export function findBuildForPackage(pkg: PackageInfo, jobs: JobSummary[]): JobSummary | undefined {
  return jobs.find((j) => jobBuiltPackage(j, pkg))
}

/** Packages a job produced, used to link a build to the catalog. */
export function packagesForJob(job: JobSummary, packages: PackageInfo[]): PackageInfo[] {
  return packages.filter((p) => jobBuiltPackage(job, p))
}
