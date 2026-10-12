import { describe, expect, it } from 'vitest'
import type { JobSummary } from '../types/api'
import { activeBuild, buildTarget, countActiveBuilds, isActive, isBuild, isFinished, isRemoval, lastFailedBuild, matchesVariant, retryRoute, variantLabel } from './jobs'

const job = (name: string, status: JobSummary['status'], extra: Partial<JobSummary> = {}): JobSummary => ({
  name,
  namespace: 'ns',
  status,
  durationSec: 60,
  variant: '',
  ...extra,
})

const removal = (name: string, status: JobSummary['status']) =>
  job(name, status, { task: 'remove-package', package: 'zfs-linux-2.4.4_7.2.8.arch1.2-1-x86_64.pkg.tar.zst' })

describe('job predicates', () => {
  it('classifies status', () => {
    expect(isActive(job('a', 'Running'))).toBe(true)
    expect(isActive(job('a', 'Pending'))).toBe(true)
    expect(isActive(job('a', 'Failed'))).toBe(false)
    expect(isFinished(job('a', 'Succeeded'))).toBe(true)
    expect(isFinished(job('a', 'Failed'))).toBe(true)
    expect(isFinished(job('a', 'Running'))).toBe(false)
  })

  it('tells builds from maintenance tasks', () => {
    expect(isBuild(job('a', 'Running'))).toBe(true)
    expect(isBuild(removal('r', 'Running'))).toBe(false)
    expect(isRemoval(removal('r', 'Running'))).toBe(true)
    expect(isRemoval(job('a', 'Running'))).toBe(false)
  })
})

describe('countActiveBuilds', () => {
  it('ignores finished jobs and maintenance tasks', () => {
    expect(countActiveBuilds([job('a', 'Running'), job('b', 'Pending'), job('c', 'Failed'), removal('r', 'Running')])).toBe(2)
  })
})

describe('activeBuild', () => {
  // newest first, like the API
  const jobs = [
    removal('r', 'Running'),
    job('lts-new', 'Running', { variant: 'lts' }),
    job('std', 'Pending'),
    job('lts-old', 'Running', { variant: 'lts' }),
  ]

  it('returns the oldest active build of the variant', () => {
    expect(activeBuild(jobs, 'lts')?.name).toBe('lts-old')
    expect(activeBuild(jobs, '')?.name).toBe('std')
  })

  it('treats a missing variant as the standard kernel', () => {
    expect(activeBuild([job('x', 'Running', { variant: undefined as unknown as string })], '')?.name).toBe('x')
  })

  it('returns undefined when nothing runs', () => {
    expect(activeBuild([job('a', 'Failed')], '')).toBeUndefined()
  })
})

describe('lastFailedBuild', () => {
  it('returns the newest finished build when it failed', () => {
    expect(lastFailedBuild([job('b', 'Failed'), job('a', 'Succeeded')], '')?.name).toBe('b')
  })

  it('returns undefined when a later build succeeded', () => {
    expect(lastFailedBuild([job('b', 'Succeeded'), job('a', 'Failed')], '')).toBeUndefined()
  })

  it('looks only at the given variant', () => {
    expect(lastFailedBuild([job('l', 'Succeeded', { variant: 'lts' }), job('a', 'Failed')], '')?.name).toBe('a')
  })

  it('skips running jobs', () => {
    expect(lastFailedBuild([job('r', 'Running'), job('a', 'Failed')], '')?.name).toBe('a')
  })

  it('ignores maintenance tasks', () => {
    expect(lastFailedBuild([removal('r', 'Succeeded'), job('a', 'Failed')], '')?.name).toBe('a')
    expect(lastFailedBuild([removal('r', 'Failed'), job('a', 'Succeeded')], '')).toBeUndefined()
  })
})

describe('variants', () => {
  it('names the kernel package', () => {
    expect(variantLabel('')).toBe('linux')
    expect(variantLabel('lts')).toBe('linux-lts')
  })

  it('treats a missing variant as the standard kernel', () => {
    expect(matchesVariant(job('a', 'Failed', { variant: undefined as unknown as string }), '')).toBe(true)
    expect(matchesVariant(job('a', 'Failed', { variant: 'lts' }), '')).toBe(false)
  })

  it('describes what a build targeted', () => {
    expect(buildTarget(job('a', 'Failed'))).toBe('linux latest')
    expect(buildTarget(job('a', 'Failed', { variant: 'lts', kernelVersion: '6.18.5-1', zfsVersion: '2.4.4' }))).toBe('linux-lts 6.18.5-1, zfs 2.4.4')
  })
})

describe('retryRoute', () => {
  it('prefills the build form with the same target', () => {
    expect(retryRoute(job('a', 'Failed', { variant: 'lts', kernelVersion: '6.18.5-1', zfsVersion: '2.4.4' }))).toEqual({
      name: 'builds',
      query: { variant: 'lts', kernel: '6.18.5-1', zfs: '2.4.4' },
    })
    expect(retryRoute(job('a', 'Failed'))).toEqual({ name: 'builds', query: {} })
  })
})
