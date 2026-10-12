import type { JobSummary } from '../types/api'

// Job lists are newest first, as served by the API.

export const isActive = (j: JobSummary) => j.status === 'Running' || j.status === 'Pending'
export const isFinished = (j: JobSummary) => j.status === 'Succeeded' || j.status === 'Failed'
/** Maintenance jobs (e.g. package removal) carry a task and never count as builds. */
export const isBuild = (j: JobSummary) => !j.task
export const isRemoval = (j: JobSummary) => j.task === 'remove-package'

const sameVariant = (a?: string, b?: string) => (a || '') === (b || '')
export const matchesVariant = (j: JobSummary, variant: string) => sameVariant(j.variant, variant)

/** Kernel variants the server builds for: "" is the standard kernel. */
export const KNOWN_VARIANTS = ['', 'lts']
export const variantLabel = (variant?: string) => (variant ? `linux-${variant}` : 'linux')

/** What a build was asked to build, e.g. "linux-lts 6.18.5-1, zfs 2.4.4". */
export function buildTarget(j: JobSummary): string {
  const kernel = `${variantLabel(j.variant)} ${j.kernelVersion || 'latest'}`
  return j.zfsVersion ? `${kernel}, zfs ${j.zfsVersion}` : kernel
}

export const countActiveBuilds = (jobs: JobSummary[]) => jobs.filter((j) => isBuild(j) && isActive(j)).length

/** The oldest active build of the kernel variant, i.e. the one that will publish first. */
export function activeBuild(jobs: JobSummary[], variant: string): JobSummary | undefined {
  return [...jobs].reverse().find((j) => isBuild(j) && isActive(j) && sameVariant(j.variant, variant))
}

/** The newest finished build of the variant, if it failed. */
export function lastFailedBuild(jobs: JobSummary[], variant: string): JobSummary | undefined {
  const last = jobs.find((j) => isBuild(j) && isFinished(j) && sameVariant(j.variant, variant))
  return last?.status === 'Failed' ? last : undefined
}

/** Builds page with the form prefilled to build the same target again. */
export function retryRoute(j: JobSummary) {
  const query: Record<string, string> = {}
  if (j.variant) query.variant = j.variant
  if (j.kernelVersion) query.kernel = j.kernelVersion
  if (j.zfsVersion) query.zfs = j.zfsVersion
  return { name: 'builds', query }
}
