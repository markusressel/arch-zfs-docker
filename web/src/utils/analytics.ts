import type { JobSummary } from '../types/api'
import { MIN_REAL_BUILD_SEC, median } from './estimate'

export interface Analytics {
  finishedCount: number
  /** Percent, or null when nothing finished yet. */
  successRate: number | null
  /** Median duration of real successful builds in seconds, 0 if unknown. */
  medianBuildSec: number
  lastSuccessAt?: string
  /** Finished jobs, oldest first. */
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

const isFinished = (b: JobSummary) => (b.status === 'Succeeded' || b.status === 'Failed') && b.durationSec > 0

/** `builds` is newest first, as served by the API. */
export function computeAnalytics(builds: JobSummary[]): Analytics {
  const finished = builds.filter(isFinished).reverse()
  const succeeded = finished.filter((b) => b.status === 'Succeeded')
  const real = succeeded.filter((b) => b.durationSec >= MIN_REAL_BUILD_SEC)
  return {
    finishedCount: finished.length,
    successRate: finished.length ? Math.round((succeeded.length / finished.length) * 100) : null,
    medianBuildSec: median(real.map((b) => b.durationSec)),
    lastSuccessAt: succeeded[succeeded.length - 1]?.endTime,
    finished,
  }
}

export function durationBuckets(finished: JobSummary[]): Bucket[] {
  return BUCKETS.map(([label, lo, hi]) => ({
    label,
    count: finished.filter((b) => b.durationSec >= lo && b.durationSec < hi).length,
  }))
}
