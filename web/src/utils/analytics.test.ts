import { describe, expect, it } from 'vitest'
import type { JobSummary } from '../types/api'
import { bucketOf, computeAnalytics, filterFromQuery, filterToQuery, durationBuckets, durationDelta, filterRuns, insights, variantHealth } from './analytics'

const job = (name: string, status: JobSummary['status'], durationSec: number, endTime?: string, extra: Partial<JobSummary> = {}): JobSummary => ({
  name,
  namespace: 'ns',
  status,
  durationSec,
  variant: '',
  endTime,
  ...extra,
})
const lts = (name: string, status: JobSummary['status'], durationSec: number) => job(name, status, durationSec, undefined, { variant: 'lts' })

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
    expect(a.failedCount).toBe(1)
    expect(a.lastSuccess?.name).toBe('b')
  })

  it('counts failures that never got to run', () => {
    const a = computeAnalytics([job('x', 'Failed', 0), ...builds])
    expect(a.failedCount).toBe(2)
    expect(a.successRate).toBe(50)
  })

  it('leaves out maintenance tasks', () => {
    const a = computeAnalytics([job('rm', 'Failed', 5), ...builds].map((b) => (b.name === 'rm' ? { ...b, task: 'remove-package' } : b)))
    expect(a.finishedCount).toBe(3)
    expect(a.finished.map((b) => b.name)).not.toContain('rm')
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

describe('durationBuckets', () => {
  it('skips jobs without a duration', () => {
    expect(durationBuckets([job('x', 'Failed', 0)]).every((b) => b.count === 0)).toBe(true)
  })

  it('maps a duration to its bucket', () => {
    expect(bucketOf(30)).toBe('<1m')
    expect(bucketOf(300)).toBe('5-15m')
    expect(bucketOf(7200)).toBe('>1h')
  })
})

describe('filterRuns', () => {
  const runs = [job('a', 'Succeeded', 600), lts('b', 'Failed', 30), lts('c', 'Succeeded', 1200)]
  const names = (f: Parameters<typeof filterRuns>[1]) => filterRuns(runs, f).map((r) => r.name)

  it('filters by variant, status and duration bucket', () => {
    expect(names({})).toEqual(['a', 'b', 'c'])
    expect(names({ variant: 'lts' })).toEqual(['b', 'c'])
    expect(names({ variant: '' })).toEqual(['a'])
    expect(names({ status: 'Failed' })).toEqual(['b'])
    expect(names({ variant: 'lts', bucket: '15-30m' })).toEqual(['c'])
  })
})

describe('variantHealth', () => {
  it('reports a failing streak since the last success', () => {
    // newest first
    const h = variantHealth([lts('f2', 'Failed', 40), job('std', 'Succeeded', 600), lts('f1', 'Failed', 50), lts('ok', 'Succeeded', 900), lts('old', 'Failed', 10)], 'lts')
    expect(h.state).toBe('failing')
    expect(h.failStreak).toBe(2)
    expect(h.lastRun?.name).toBe('f2')
    expect(h.lastFailure?.name).toBe('f2')
    expect(h.lastSuccess?.name).toBe('ok')
    expect(h.runs).toBe(4)
    expect(h.successRate).toBe(25)
    expect(h.recent.map((r) => r.name)).toEqual(['old', 'ok', 'f1', 'f2'])
  })

  it('is healthy after a success and building while a job runs', () => {
    expect(variantHealth([job('ok', 'Succeeded', 600), job('f', 'Failed', 40)], '').state).toBe('healthy')
    const h = variantHealth([job('run', 'Running', 5), job('f', 'Failed', 40)], '')
    expect(h.state).toBe('building')
    expect(h.active?.name).toBe('run')
  })

  it('is idle without builds and ignores maintenance tasks', () => {
    const h = variantHealth([job('rm', 'Failed', 5, undefined, { task: 'remove-package' })], '')
    expect(h.state).toBe('idle')
    expect(h.successRate).toBeNull()
  })
})

describe('durationDelta', () => {
  it('compares a real successful build with the median', () => {
    expect(durationDelta(job('a', 'Succeeded', 900), 600)).toBe(50)
    expect(durationDelta(job('a', 'Succeeded', 300), 600)).toBe(-50)
  })

  it('is null when there is nothing to compare', () => {
    expect(durationDelta(job('a', 'Failed', 900), 600)).toBeNull()
    expect(durationDelta(job('a', 'Succeeded', 30), 600)).toBeNull()
    expect(durationDelta(job('a', 'Succeeded', 900), 0)).toBeNull()
  })
})

describe('insights', () => {
  it('flags failing variants first, with the newest failure', () => {
    const list = insights([job('run', 'Running', 5), lts('f2', 'Failed', 40), lts('f1', 'Failed', 50), lts('ok', 'Succeeded', 900)])
    expect(list.map((i) => i.kind)).toEqual(['failing', 'building'])
    const failing = list[0]!
    expect(failing.kind === 'failing' && failing.count).toBe(2)
    expect(failing.variant).toBe('lts')
    expect(failing.job.name).toBe('f2')
  })

  it('flags a build that took much longer than usual', () => {
    const list = insights([job('slow', 'Succeeded', 1800), job('a', 'Succeeded', 600), job('b', 'Succeeded', 660), job('c', 'Succeeded', 540)])
    expect(list).toHaveLength(1)
    expect(list[0]).toMatchObject({ kind: 'slow', variant: '', percent: 186, medianSec: 630 })
    expect(list[0]!.job.name).toBe('slow')
  })

  it('needs enough history to call a build slow', () => {
    expect(insights([job('slow', 'Succeeded', 1800), job('a', 'Succeeded', 600)])).toEqual([])
  })

  it('only looks at the requested variants', () => {
    expect(insights([lts('f', 'Failed', 40)], [''])).toEqual([])
  })
})

describe('filter query', () => {
  it('round trips through the URL', () => {
    for (const f of [{}, { variant: '' }, { variant: 'lts', status: 'Failed' as const, bucket: '5-15m' }]) {
      expect(filterFromQuery(filterToQuery(f))).toEqual(f)
    }
  })

  it('ignores unknown values', () => {
    expect(filterFromQuery({ variant: ['lts'], status: 'Running', bucket: 'nope' })).toEqual({})
  })
})
