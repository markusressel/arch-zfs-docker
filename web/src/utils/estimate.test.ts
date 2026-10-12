import { describe, expect, it } from 'vitest'
import type { JobSummary } from '../types/api'
import { buildEta, estimateDuration, etaText, median, progressPercent } from './estimate'

const job = (over: Partial<JobSummary>): JobSummary => ({
  name: 'j',
  namespace: 'ns',
  status: 'Succeeded',
  durationSec: 600,
  variant: '',
  ...over,
})

describe('median', () => {
  it('handles odd, even and empty input', () => {
    expect(median([3, 1, 2])).toBe(2)
    expect(median([1, 2, 3, 4])).toBe(2.5)
    expect(median([])).toBe(0)
  })
})

describe('estimateDuration', () => {
  it('ignores cache hits and failures', () => {
    const builds = [job({ durationSec: 20 }), job({ status: 'Failed', durationSec: 9999 }), job({ durationSec: 900 })]
    expect(estimateDuration(builds, '')).toBe(900)
  })

  it('prefers the same variant, falling back to any', () => {
    const builds = [job({ variant: 'lts', durationSec: 300 }), job({ variant: '', durationSec: 1000 })]
    expect(estimateDuration(builds, 'lts')).toBe(300)
    expect(estimateDuration(builds, 'zen')).toBe(650)
  })

  it('returns 0 without history', () => {
    expect(estimateDuration([], '')).toBe(0)
  })
})

describe('etaText / progressPercent', () => {
  it('formats remaining time', () => {
    expect(etaText(60, 600)).toBe('~09:00 left')
    expect(etaText(700, 600)).toBe('taking longer than usual')
    expect(etaText(10, 0)).toBe('')
  })

  it('caps progress below 100', () => {
    expect(progressPercent(300, 600)).toBe(50)
    expect(progressPercent(900, 600)).toBe(99)
    expect(progressPercent(10, 0)).toBe(0)
  })
})

describe('buildEta', () => {
  const now = new Date('2026-10-12T12:00:00Z').getTime()
  const history = [job({ durationSec: 600 })]

  it('counts down to the expected finish time', () => {
    const running = job({ status: 'Running', startTime: '2026-10-12T11:55:00Z' })
    const eta = buildEta(running, history, now)
    expect(eta.elapsedSec).toBe(300)
    expect(eta.remainingSec).toBe(300)
    expect(eta.finishAt).toBe(now + 300_000)
    expect(eta.overdue).toBe(false)
  })

  it('reports overdue builds without a finish time', () => {
    const running = job({ status: 'Running', startTime: '2026-10-12T11:40:00Z' })
    const eta = buildEta(running, history, now)
    expect(eta.overdue).toBe(true)
    expect(eta.finishAt).toBeNull()
  })

  it('has no estimate without history', () => {
    const eta = buildEta(job({ status: 'Running', startTime: '2026-10-12T11:55:00Z' }), [], now)
    expect(eta.estimateSec).toBe(0)
    expect(eta.finishAt).toBeNull()
    expect(eta.overdue).toBe(false)
  })
})
