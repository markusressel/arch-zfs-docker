import type { JobSummary } from '../types/api'
import { formatDuration } from './format'

/** Builds finishing faster than this were cache hits / no-ops and say nothing about real build time. */
export const MIN_REAL_BUILD_SEC = 120

const RECENT_SAMPLE_SIZE = 5

export function median(values: number[]): number {
  if (values.length === 0) return 0
  const sorted = [...values].sort((a, b) => a - b)
  const mid = Math.floor(sorted.length / 2)
  return sorted.length % 2 ? sorted[mid]! : (sorted[mid - 1]! + sorted[mid]!) / 2
}

/**
 * Estimates the duration (seconds) of a build from recent real successful builds,
 * preferring jobs of the same kernel variant. `builds` must be newest first. Returns 0 when unknown.
 */
export function estimateDuration(builds: JobSummary[], variant: string): number {
  const real = builds.filter((b) => b.status === 'Succeeded' && b.durationSec >= MIN_REAL_BUILD_SEC)
  const sameVariant = real.filter((b) => (b.variant || '') === (variant || ''))
  const pool = (sameVariant.length > 0 ? sameVariant : real).slice(0, RECENT_SAMPLE_SIZE)
  return median(pool.map((b) => b.durationSec))
}

export function etaText(elapsedSec: number, estimateSec: number): string {
  if (!estimateSec) return ''
  const remaining = estimateSec - elapsedSec
  return remaining > 0 ? `~${formatDuration(remaining)} left` : 'taking longer than usual'
}

/** Progress in percent, capped below 100 as long as the job is still running. */
export function progressPercent(elapsedSec: number, estimateSec: number): number {
  if (!estimateSec) return 0
  return Math.min(99, (elapsedSec / estimateSec) * 100)
}
