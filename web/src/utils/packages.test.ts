import { describe, expect, it } from 'vitest'
import type { JobSummary, PackageInfo } from '../types/api'
import { filterPackages, findBuildForPackage, packagesForJob } from './packages'

const pkg = (packageName: string, kernelVersion: string | undefined, version = '2.4.4'): PackageInfo => ({
  filename: `${packageName}-${version}-${kernelVersion}.pkg.tar.zst`,
  packageName,
  version,
  kernelVersion,
  arch: 'x86_64',
  sizeBytes: 1,
  sizeHuman: '1 B',
  modTime: '',
  downloadUrl: '/x',
})

const job = (name: string, over: Partial<JobSummary> = {}): JobSummary => ({
  name,
  namespace: 'arch-repo',
  status: 'Succeeded',
  durationSec: 1,
  variant: '',
  kernelVersion: '7.2.9.arch1-1',
  ...over,
})

describe('filterPackages', () => {
  it('matches name, version and kernel', () => {
    const list = [pkg('zfs-linux', '7.2.9.arch1.1'), pkg('zfs-utils', undefined)]
    expect(filterPackages(list, 'utils')).toHaveLength(1)
    expect(filterPackages(list, '7.2.9')).toHaveLength(1)
  })
})

describe('build <-> package matching', () => {
  it('matches kernel across dash/dot notation and variant', () => {
    const jobs = [job('lts-job', { variant: 'lts', kernelVersion: '6.18.55-1' }), job('std-job')]
    expect(findBuildForPackage(pkg('zfs-linux', '7.2.9.arch1.1'), jobs)?.name).toBe('std-job')
    expect(findBuildForPackage(pkg('zfs-linux-headers', '7.2.9.arch1.1'), jobs)?.name).toBe('std-job')
    expect(findBuildForPackage(pkg('zfs-linux-lts', '6.18.55.1'), jobs)?.name).toBe('lts-job')
  })

  it('ignores failed jobs, "latest" jobs, kernel independent packages and other ZFS versions', () => {
    const p = pkg('zfs-linux', '7.2.9.arch1.1')
    expect(findBuildForPackage(p, [job('x', { status: 'Failed' })])).toBeUndefined()
    expect(findBuildForPackage(p, [job('x', { kernelVersion: '' })])).toBeUndefined()
    expect(findBuildForPackage(pkg('zfs-utils', undefined), [job('x')])).toBeUndefined()
    expect(findBuildForPackage(p, [job('x', { zfsVersion: '2.3.9' })])).toBeUndefined()
  })

  it('lists the packages a job produced', () => {
    const list = [pkg('zfs-linux', '7.2.9.arch1.1'), pkg('zfs-linux-headers', '7.2.9.arch1.1'), pkg('zfs-linux', '7.2.8.arch1.2')]
    expect(packagesForJob(job('j'), list)).toHaveLength(2)
  })
})
