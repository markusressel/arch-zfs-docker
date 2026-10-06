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
    expect(wrapper.text()).toContain('zfs-build-1')
    expect(wrapper.text()).toContain('1 packages')
    expect(wrapper.text()).toContain('SigLevel = Optional TrustAll')
    expect(wrapper.findAll('#kernel-versions option')).toHaveLength(2)
    expect(wrapper.findAll('#zfs-versions option')).toHaveLength(3)
    // Check Now only lives on the Upstream Sync card
    expect(wrapper.findAll('button').filter((b) => b.text() === 'Check Now')).toHaveLength(1)
    wrapper.unmount()
  })

  it('switches to analytics and shows statistics', async () => {
    const wrapper = await mountApp()
    await flushPromises()

    await wrapper.findAll('nav a').find((a) => a.text() === 'Analytics')!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Success rate')
    expect(wrapper.text()).toContain('50%')
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
    const wrapper = await mountApp()
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
