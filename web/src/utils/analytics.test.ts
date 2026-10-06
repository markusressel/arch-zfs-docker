import { describe, expect, it } from 'vitest'
import type { JobSummary } from '../types/api'
import { computeAnalytics, durationBuckets } from './analytics'

const job = (name: string, status: JobSummary['status'], durationSec: number, endTime?: string): JobSummary => ({
  name,
  namespace: 'ns',
  status,
  durationSec,
  variant: '',
  endTime,
})

// newest first, like the API
const builds = [
  job('c', 'Failed', 30),
  job('b', 'Succeeded', 600, '2026-01-02T00:00:00Z'),
  job('a', 'Succeeded', 1200, '2026-01-01T00:00:00Z'),
  job('run', 'Running', 5),
]

describe('computeAnalytics', () => {
  it('summarises finished jobs only', () => {
    const a = computeAnalytics(builds)
    expect(a.finishedCount).toBe(3)
    expect(a.successRate).toBe(67)
    expect(a.medianBuildSec).toBe(900)
    expect(a.lastSuccessAt).toBe('2026-01-02T00:00:00Z')
    expect(a.finished.map((b) => b.name)).toEqual(['a', 'b', 'c'])
  })

  it('handles empty input', () => {
    const a = computeAnalytics([])
    expect(a.successRate).toBeNull()
    expect(a.medianBuildSec).toBe(0)
  })
})

describe('durationBuckets', () => {
  it('counts jobs per bucket', () => {
    const counts = Object.fromEntries(durationBuckets(computeAnalytics(builds).finished).map((b) => [b.label, b.count]))
    expect(counts['<1m']).toBe(1)
    expect(counts['5-15m']).toBe(1)
    expect(counts['15-30m']).toBe(1)
  })
})
