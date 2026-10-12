import type { JobSummary } from '../types/api'
import { MIN_REAL_BUILD_SEC, median } from './estimate'
import { KNOWN_VARIANTS, isActive, isBuild, isFinished, matchesVariant } from './jobs'

// Job lists passed in are newest first, as served by the API; maintenance jobs are always left out.

export interface Analytics {
  finishedCount: number
  failedCount: number
  /** Percent, or null when nothing finished yet. */
  successRate: number | null
  /** Median duration of real successful builds in seconds, 0 if unknown. */
  medianBuildSec: number
  lastSuccessAt?: string
  lastSuccess?: JobSummary
  /** Finished builds, oldest first. */
  finished: JobSummary[]
}

export interface Bucket {
  label: string
  count: number
}

const BUCKETS: Array<[label: string, lo: number, hi: number]> = [
  ['<1m', 0, 60],
  ['1-5m', 60, 300],
  ['5-15m', 300, 900],
  ['15-30m', 900, 1800],
  ['30-60m', 1800, 3600],
  ['>1h', 3600, Infinity],
]

/** Newest first. */
const finishedBuilds = (builds: JobSummary[]) => builds.filter((b) => isBuild(b) && isFinished(b))
const isReal = (b: JobSummary) => b.status === 'Succeeded' && b.durationSec >= MIN_REAL_BUILD_SEC
const percent = (part: number, total: number) => (total ? Math.round((part / total) * 100) : null)

export function computeAnalytics(builds: JobSummary[]): Analytics {
  const finished = finishedBuilds(builds).reverse()
  const succeeded = finished.filter((b) => b.status === 'Succeeded')
  const lastSuccess = succeeded[succeeded.length - 1]
  return {
    finishedCount: finished.length,
    failedCount: finished.length - succeeded.length,
    successRate: percent(succeeded.length, finished.length),
    medianBuildSec: median(succeeded.filter(isReal).map((b) => b.durationSec)),
    lastSuccessAt: lastSuccess?.endTime,
    lastSuccess,
    finished,
  }
}

export function bucketOf(durationSec: number): string {
  return BUCKETS.find(([, lo, hi]) => durationSec >= lo && durationSec < hi)![0]
}

/** Jobs without a duration never ran and are left out. */
export function durationBuckets(finished: JobSummary[]): Bucket[] {
  const timed = finished.filter((b) => b.durationSec > 0)
  return BUCKETS.map(([label]) => ({ label, count: timed.filter((b) => bucketOf(b.durationSec) === label).length }))
}

export type RunStatus = 'Succeeded' | 'Failed'

export interface RunFilter {
  /** Undefined for all variants. */
  variant?: string
  status?: RunStatus
  /** Duration bucket label. */
  bucket?: string
}

// In the URL the standard kernel is "linux", so every filter has a readable, non-empty value.
const STATUSES: RunStatus[] = ['Succeeded', 'Failed']
const str = (v: unknown) => (typeof v === 'string' ? v : undefined)

export function filterFromQuery(query: Record<string, unknown>): RunFilter {
  const f: RunFilter = {}
  const variant = str(query.variant)
  if (variant === 'linux') f.variant = ''
  else if (variant && KNOWN_VARIANTS.includes(variant)) f.variant = variant
  const status = str(query.status)
  if (STATUSES.includes(status as RunStatus)) f.status = status as RunStatus
  const bucket = str(query.bucket)
  if (BUCKETS.some(([label]) => label === bucket)) f.bucket = bucket
  return f
}

export function filterToQuery(f: RunFilter): Record<string, string> {
  const query: Record<string, string> = {}
  if (f.variant !== undefined) query.variant = f.variant || 'linux'
  if (f.status) query.status = f.status
  if (f.bucket) query.bucket = f.bucket
  return query
}

export function filterRuns(runs: JobSummary[], f: RunFilter): JobSummary[] {
  return runs.filter(
    (r) =>
      (f.variant === undefined || matchesVariant(r, f.variant)) &&
      (!f.status || r.status === f.status) &&
      (!f.bucket || (r.durationSec > 0 && bucketOf(r.durationSec) === f.bucket)),
  )
}

/** Signed percentage a real successful build differs from the median, null when there is nothing to compare. */
export function durationDelta(job: JobSummary, medianSec: number): number | null {
  if (!medianSec || !isReal(job)) return null
  return Math.round(((job.durationSec - medianSec) / medianSec) * 100)
}

const RECENT_RUNS = 12

export type HealthState = 'healthy' | 'failing' | 'building' | 'idle'

export interface VariantHealth {
  variant: string
  state: HealthState
  runs: number
  successRate: number | null
  /** Failed builds since the last success. */
  failStreak: number
  lastRun?: JobSummary
  lastSuccess?: JobSummary
  lastFailure?: JobSummary
  active?: JobSummary
  medianSec: number
  /** Latest finished builds, oldest first. */
  recent: JobSummary[]
}

export function variantHealth(builds: JobSummary[], variant: string): VariantHealth {
  const own = builds.filter((b) => matchesVariant(b, variant))
  const finished = finishedBuilds(own)
  const succeeded = finished.filter((b) => b.status === 'Succeeded')
  const streak = finished.findIndex((b) => b.status === 'Succeeded')
  const failStreak = streak === -1 ? finished.length : streak
  const active = [...own].reverse().find((b) => isBuild(b) && isActive(b))
  return {
    variant,
    state: active ? 'building' : failStreak > 0 ? 'failing' : finished.length ? 'healthy' : 'idle',
    runs: finished.length,
    successRate: percent(succeeded.length, finished.length),
    failStreak,
    lastRun: finished[0],
    lastSuccess: succeeded[0],
    lastFailure: finished.find((b) => b.status === 'Failed'),
    active,
    medianSec: median(succeeded.filter(isReal).map((b) => b.durationSec)),
    recent: finished.slice(0, RECENT_RUNS).reverse(),
  }
}

/** A build this much slower than the median of earlier ones is called out. */
const SLOW_FACTOR = 1.5
/** Earlier real builds needed before calling one slow. */
const SLOW_MIN_HISTORY = 3

export type Insight =
  | { kind: 'failing'; severity: 'error'; variant: string; job: JobSummary; count: number; lastSuccess?: JobSummary }
  | { kind: 'slow'; severity: 'warning'; variant: string; job: JobSummary; percent: number; medianSec: number }
  | { kind: 'building'; severity: 'info'; variant: string; job: JobSummary }

const SEVERITY_ORDER = { error: 0, warning: 1, info: 2 }

/** Things worth acting on, most severe first. */
export function insights(builds: JobSummary[], variants: string[] = KNOWN_VARIANTS): Insight[] {
  const out: Insight[] = []
  for (const variant of variants) {
    const h = variantHealth(builds, variant)
    if (h.failStreak > 0 && h.lastFailure) {
      out.push({ kind: 'failing', severity: 'error', variant, job: h.lastFailure, count: h.failStreak, lastSuccess: h.lastSuccess })
    }
    const last = h.lastRun
    // Compared with the same median the build list shows; enough earlier builds keep one outlier from setting it.
    const earlier = h.recent.slice(0, -1).filter(isReal).length
    if (last && isReal(last) && earlier >= SLOW_MIN_HISTORY && last.durationSec > h.medianSec * SLOW_FACTOR) {
      out.push({ kind: 'slow', severity: 'warning', variant, job: last, percent: durationDelta(last, h.medianSec)!, medianSec: h.medianSec })
    }
    if (h.active) out.push({ kind: 'building', severity: 'info', variant, job: h.active })
  }
  return out.sort((a, b) => SEVERITY_ORDER[a.severity] - SEVERITY_ORDER[b.severity])
}
