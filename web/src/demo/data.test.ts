import { describe, expect, it } from 'vitest'
import type { JobSummary, RepoSummary, UpstreamStatus } from '../types/api'
import { demoFetch, demoResponses } from './data'

describe('demo data', () => {
  const now = Date.parse('2026-11-20T12:00:00Z')
  const data = demoResponses(now)

  it('shows a healthy repository', () => {
    const upstream = data['/api/upstream'] as UpstreamStatus
    expect(Object.keys(upstream)).toEqual(['linux', 'linux-lts'])
    for (const v of Object.values(upstream)) {
      expect(v.upToDate).toBe(true)
      expect(v.localKernel).toBe(v.upstreamKernel.replace('-', '.'))
    }
    const builds = data['/api/builds'] as JobSummary[]
    expect(builds.filter((b) => b.status === 'Failed')).toHaveLength(1)
    expect(builds.every((b) => Date.parse(b.endTime!) < now)).toBe(true)
  })

  it('has packages for the newest kernels', () => {
    const repo = data['/api/packages'] as RepoSummary
    expect(repo.packageCount).toBe(repo.packages.length)
    expect(repo.packages.map((p) => p.kernelVersion)).toContain('7.2.14.arch1.1')
    expect(repo.packages.map((p) => p.kernelVersion)).toContain('6.18.64.1')
  })

  it('answers GET requests and refuses changes', async () => {
    expect((await demoFetch('/api/settings')).ok).toBe(true)
    expect((await demoFetch('/api/kernels?variant=lts')).ok).toBe(true)
    expect((await demoFetch('/api/nope')).status).toBe(404)
    expect((await demoFetch('/api/builds', { method: 'POST' })).status).toBe(403)
  })
})
