import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App.vue'
import { createAppRouter } from './router'

const now = new Date().toISOString()

const responses: Record<string, unknown> = {
  '/api/settings': { autoCheckInterval: '6h', buildNode: '', repoName: 'zfslocal', namespace: 'arch-repo', listenAddr: ':8080', repoDir: '/repo' },
  '/api/upstream': {
    linux: { variant: '', kernelPkg: 'linux', upstreamKernel: '7.2.8.arch1-2', localKernel: '7.2.7.arch1.1', upToDate: false, checkedAt: now },
  },
  '/api/builds': [
    { name: 'zfs-build-1', namespace: 'arch-repo', status: 'Succeeded', durationSec: 600, variant: '', startTime: now, endTime: now },
    { name: 'zfs-build-2', namespace: 'arch-repo', status: 'Failed', durationSec: 40, variant: '', startTime: now },
  ],
  '/api/packages': {
    packages: [{ filename: 'zfs-linux-1.pkg.tar.zst', packageName: 'zfs-linux', version: '2.4.4', kernelVersion: '7.2.7.arch1.1', sizeBytes: 2048, sizeHuman: '2.00 KiB', modTime: now, downloadUrl: '/x' }],
  },
  '/api/kernels?variant=': { versions: ['7.2.8.arch1-2', '7.2.7.arch1-1'] },
  '/api/zfs-versions': { versions: ['2.4.4', '2.3.9', '2.2.11'] },
}

class FakeEventSource {
  static last: FakeEventSource
  onmessage: ((e: { data: string }) => void) | null = null
  onerror: (() => void) | null = null
  onopen: (() => void) | null = null
  url: string
  constructor(url: string) {
    this.url = url
    FakeEventSource.last = this
  }
  addEventListener() {}
  close() {}
}

async function mountApp(hash = '#/') {
  window.location.hash = hash
  const router = createAppRouter()
  router.push(hash.slice(1))
  await router.isReady()
  return mount(App, { attachTo: document.body, global: { plugins: [router] } })
}

describe('App', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn(async (path: string) => ({ ok: true, status: 200, json: async () => responses[path], text: async () => '' })))
    vi.stubGlobal('EventSource', FakeEventSource)
  })
  afterEach(() => vi.unstubAllGlobals())

  it('renders the dashboard from the API', async () => {
    const wrapper = await mountApp()
    await flushPromises()

    expect(wrapper.text()).toContain('Build needed')
    expect(wrapper.text()).toContain('7.2.8.arch1-2')
    expect(wrapper.text()).toContain('1 packages')
    expect(wrapper.text()).toContain('SigLevel = Optional TrustAll')
    expect(wrapper.find('a[href="/zfslocal/x86_64/"]').text()).toBe('Browse files')
    // Build controls live on the Builds page
    expect(wrapper.text()).not.toContain('zfs-build-1')
    expect(wrapper.find('#kernel-versions').exists()).toBe(false)
    // Check Now only lives on the Upstream Sync card
    expect(wrapper.findAll('button').filter((b) => b.text() === 'Check Now')).toHaveLength(1)
    wrapper.unmount()
  })

  it('shows when the missing package will be available', async () => {
    const original = responses['/api/builds']
    const started = new Date(Date.now() - 5 * 60_000).toISOString()
    responses['/api/builds'] = [
      { name: 'zfs-build-run', namespace: 'arch-repo', status: 'Running', durationSec: 300, variant: '', startTime: started },
      { name: 'zfs-build-1', namespace: 'arch-repo', status: 'Succeeded', durationSec: 600, variant: '', startTime: now, endTime: now },
    ]
    try {
      const wrapper = await mountApp()
      await flushPromises()
      expect(wrapper.text()).toContain('Building')
      expect(wrapper.text()).toContain('Package available')
      expect(wrapper.text()).toMatch(/in about \d+ min \(around/)
      expect(wrapper.text()).toContain('updates automatically')
      wrapper.unmount()
    } finally {
      responses['/api/builds'] = original
    }
  })

  it('explains a kernel that OpenZFS cannot build against yet', async () => {
    const original = responses['/api/upstream']
    responses['/api/upstream'] = {
      linux: {
        variant: '',
        kernelPkg: 'linux',
        upstreamKernel: '7.3.1.arch1-1',
        localKernel: '7.2.9.arch1.1',
        upToDate: false,
        checkedAt: now,
        compat: {
          state: 'incompatible',
          zfsVersion: '2.4.4',
          linuxMax: '7.2',
          reason: 'OpenZFS does not support this kernel version',
          job: 'zfs-build-2',
          issues: [{ number: 18204, title: 'Linux 7.3 compat', url: 'https://example.test/18204', state: 'open' }],
          issuesUrl: 'https://example.test/search',
        },
      },
    }
    try {
      const wrapper = await mountApp()
      await flushPromises()
      expect(wrapper.text()).toContain('Waiting for ZFS')
      expect(wrapper.text()).toContain('does not build against kernel 7.3.1.arch1-1')
      expect(wrapper.text()).toContain('#18204 Linux 7.3 compat')
      expect(wrapper.text()).toContain('Show failed build log')
      wrapper.unmount()
    } finally {
      responses['/api/upstream'] = original
    }
  })

  it('explains the automatic sync when no packages exist yet', async () => {
    const original = responses['/api/packages']
    responses['/api/packages'] = { packages: [] }
    try {
      const wrapper = await mountApp()
      await flushPromises()
      expect(wrapper.text()).toContain('No packages published yet')
      expect(wrapper.text()).toContain('builds missing packages automatically')
      expect(wrapper.text()).toContain('manual builds are usually not necessary')
      wrapper.unmount()
    } finally {
      responses['/api/packages'] = original
    }
  })

  it('points at the failed build when the package is still missing', async () => {
    const original = responses['/api/builds']
    responses['/api/builds'] = [{ name: 'zfs-build-9', namespace: 'arch-repo', status: 'Failed', durationSec: 9, variant: '', startTime: now, endTime: now }]
    try {
      const wrapper = await mountApp()
      await flushPromises()
      expect(wrapper.text()).toContain('The last build failed')
      expect(wrapper.text()).toContain('Show build log')
      wrapper.unmount()
    } finally {
      responses['/api/builds'] = original
    }
  })

  it('does not treat package removals as builds', async () => {
    const original = responses['/api/builds']
    responses['/api/builds'] = [
      { name: 'zfs-build-rm2', namespace: 'arch-repo', status: 'Running', durationSec: 2, variant: '', startTime: now, task: 'remove-package', package: 'x.pkg.tar.zst' },
      { name: 'zfs-build-rm1', namespace: 'arch-repo', status: 'Succeeded', durationSec: 3, variant: '', startTime: now, endTime: now, task: 'remove-package', package: 'y.pkg.tar.zst' },
      { name: 'zfs-build-9', namespace: 'arch-repo', status: 'Failed', durationSec: 9, variant: '', startTime: now, endTime: now },
    ]
    try {
      const wrapper = await mountApp()
      await flushPromises()
      expect(wrapper.find('.tabs .badge').exists()).toBe(false)
      expect(wrapper.text()).not.toContain('Building')
      // The succeeded removal must not hide the failed build.
      expect(wrapper.text()).toContain('The last build failed')
      wrapper.unmount()
    } finally {
      responses['/api/builds'] = original
    }
  })

  it('asks for confirmation before removing a package and explains the rebuild', async () => {
    const calls: string[] = []
    const base = globalThis.fetch as (path: string, init?: RequestInit) => Promise<unknown>
    vi.stubGlobal('fetch', vi.fn(async (path: string, init?: RequestInit) => {
      if (init?.method === 'DELETE') {
        calls.push(path)
        return { ok: true, status: 202, json: async () => ({ name: 'zfs-build-rm', status: 'Running', variant: '', durationSec: 0, namespace: 'x' }), text: async () => '' }
      }
      return base(path, init)
    }))
    const original = responses['/api/upstream']
    responses['/api/upstream'] = {
      linux: { variant: '', kernelPkg: 'linux', upstreamKernel: '7.2.7.arch1-1', localKernel: '7.2.7.arch1.1', upToDate: true, checkedAt: now },
    }
    try {
      const wrapper = await mountApp()
      await flushPromises()

      await wrapper.findAll('button').find((b) => b.text() === 'Remove')!.trigger('click')
      await flushPromises()
      expect(document.body.textContent).toContain('Remove package from the repository?')
      expect(document.body.textContent).toContain('automatically start a new build')
      expect(calls).toHaveLength(0) // nothing happens before confirming

      const confirmButton = Array.from(document.body.querySelectorAll('button')).find((b) => b.textContent?.trim() === 'Remove package')!
      confirmButton.click()
      await flushPromises()
      expect(calls).toEqual(['/api/packages/zfs-linux-1.pkg.tar.zst'])
      wrapper.unmount()
    } finally {
      responses['/api/upstream'] = original
    }
  })

  it('links the build chip to the builds page with the kernel prefilled', async () => {
    const wrapper = await mountApp()
    await flushPromises()

    await wrapper.find('a.pill-link').trigger('click')
    await flushPromises()
    expect((wrapper.find('#kernel-input').element as HTMLInputElement).value).toBe('7.2.8.arch1-2')
    wrapper.unmount()
  })

  it('renders the builds page from the API', async () => {
    const wrapper = await mountApp('#/builds')
    await flushPromises()

    expect(wrapper.text()).toContain('zfs-build-1')
    expect(wrapper.findAll('#kernel-versions option')).toHaveLength(2)
    expect(wrapper.findAll('#zfs-versions option')).toHaveLength(3)
    expect(wrapper.text()).not.toContain('Upstream Sync')
    wrapper.unmount()
  })

  it('switches to analytics and shows statistics', async () => {
    const wrapper = await mountApp()
    await flushPromises()

    await wrapper.findAll('nav a').find((a) => a.text() === 'Analytics')!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Success rate')
    expect(wrapper.text()).toContain('50%')
    // The newest build succeeded, so nothing needs attention
    expect(wrapper.text()).toContain('All good')
    wrapper.unmount()
  })

  it('filters analytics by clicking and links back to logs and builds', async () => {
    const original = responses['/api/builds']
    responses['/api/builds'] = [
      { name: 'zfs-build-3', namespace: 'arch-repo', status: 'Failed', durationSec: 40, variant: 'lts', kernelVersion: '6.18.5-1', startTime: now, endTime: now },
      ...(original as unknown[]),
    ]
    try {
      const wrapper = await mountApp('#/analytics')
      await flushPromises()
      const router = wrapper.vm.$router

      // Needs attention: the failing lts build, with a retry prefilled for the same kernel
      expect(wrapper.text()).toContain('linux-lts: the latest build failed')
      const retry = wrapper.findAll('a').find((a) => a.text() === 'Retry build')!
      expect(retry.attributes('href')).toBe('#/builds?variant=lts&kernel=6.18.5-1')

      // The failed builds KPI filters the list and is kept in the URL
      await wrapper.findAll('button.kpi').find((b) => b.text().includes('Failed builds'))!.trigger('click')
      await flushPromises()
      expect(router.currentRoute.value.query).toEqual({ status: 'Failed' })
      const rows = () => wrapper.findAll('tbody tr.row').map((r) => r.find('code').text())
      expect(rows()).toEqual(['zfs-build-3', 'zfs-build-2'])

      // Show runs on a kernel card narrows it further
      await wrapper.findAll('button').find((b) => b.text() === 'Show runs')!.trigger('click')
      await flushPromises()
      expect(router.currentRoute.value.query).toEqual({ status: 'Failed', variant: 'linux' })
      expect(rows()).toEqual(['zfs-build-2'])

      // A row opens the log
      await wrapper.find('tbody tr.row').trigger('click')
      expect(FakeEventSource.last.url).toBe('/api/builds/zfs-build-2/logs')

      await wrapper.findAll('button').find((b) => b.text() === 'Reset filters')!.trigger('click')
      await flushPromises()
      expect(router.currentRoute.value.query).toEqual({})
      expect(rows()).toHaveLength(3)
      wrapper.unmount()
    } finally {
      responses['/api/builds'] = original
    }
  })

  it('restores analytics filters from the URL', async () => {
    const wrapper = await mountApp('#/analytics?variant=lts')
    await flushPromises()
    expect(wrapper.text()).toContain('No finished builds match the filters')
    wrapper.unmount()
  })

  it('shows the settings view when opened by URL', async () => {
    const wrapper = await mountApp('#/settings')
    await flushPromises()

    expect(wrapper.text()).toContain('Dynamic Service Settings')
    expect(wrapper.text()).not.toContain('Upstream Sync')
    wrapper.unmount()
  })

  it('opens the log modal and streams lines', async () => {
    const wrapper = await mountApp('#/builds')
    await flushPromises()

    await wrapper.findAll('button').find((b) => b.text() === 'Logs')!.trigger('click')
    expect(FakeEventSource.last.url).toBe('/api/builds/zfs-build-1/logs')
    FakeEventSource.last.onmessage?.({ data: '==> hello from build' })
    await new Promise((r) => requestAnimationFrame(() => r(null)))
    await flushPromises()
    expect(document.body.textContent).toContain('==> hello from build')
    wrapper.unmount()
  })
})
