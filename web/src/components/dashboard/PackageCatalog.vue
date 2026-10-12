<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { deletePackage } from '../../api/repo'
import { useBuilds } from '../../composables/useBuilds'
import { useDateFormat } from '../../composables/useDateFormat'
import { useDialogs } from '../../composables/useDialogs'
import { useLogViewer } from '../../composables/useLogViewer'
import { usePackages } from '../../composables/usePackages'
import { useSettings } from '../../composables/useSettings'
import { useUpstream } from '../../composables/useUpstream'
import { describeInterval, formatBytes } from '../../utils/format'
import type { PackageInfo } from '../../types/api'
import { isActive, isRemoval } from '../../utils/jobs'
import { filterPackages, findBuildForPackage, removalImpact, type RemovalImpact } from '../../utils/packages'
import AppButton from '../ui/AppButton.vue'
import AppCard from '../ui/AppCard.vue'

const { packages, refresh } = usePackages()
const { builds, refresh: refreshBuilds } = useBuilds()
const { variants } = useUpstream()
const { confirm, alert } = useDialogs()
const logViewer = useLogViewer()
const dates = useDateFormat()
const { settings } = useSettings()
// Unknown (settings not loaded yet) counts as enabled, the default.
const autoSync = computed(() => {
  const interval = settings.value?.autoCheckInterval
  return interval === undefined || !['', '0', '0s'].includes(interval)
})

const consequences: Record<RemovalImpact, string> = {
  rebuild:
    'This is the package for the current upstream kernel. The upstream sync will notice it is missing and automatically start a new build right away, which takes several minutes and re-triggers the auto-sync.',
  headers:
    'These are the headers of the current upstream kernel. They are not rebuilt on their own: the upstream sync only rebuilds when the main package is missing, so they come back with the next build of that package.',
  utils: 'zfs-utils is kernel independent. It is not rebuilt by the sync on its own, but the next kernel build recreates it.',
  older: 'This is not the build for the current upstream kernel, so it is not rebuilt automatically. Systems still on that kernel can no longer install it.',
}

/** Remove jobs in flight, by package file name. */
const removing = computed(() => new Set(builds.value.filter((b) => isRemoval(b) && isActive(b)).map((b) => b.package)))

async function onRemove(p: PackageInfo) {
  const ok = await confirm({
    title: 'Remove package from the repository?',
    message: `${consequences[removalImpact(p, variants.value)]}\n\nThe file is deleted from the repository and the pacman database is regenerated. Clients that already installed it are not affected.`,
    detail: p.filename,
    confirmLabel: 'Remove package',
    danger: true,
  })
  if (!ok) return
  try {
    await deletePackage(p.filename)
    await refreshBuilds()
  } catch (err) {
    await alert('Failed to remove package', err instanceof Error ? err.message : String(err))
  }
}

// The filter lives in the URL: builds and the upstream sync link here with ?q=<kernel> to show their packages.
const route = useRoute()
const router = useRouter()
const search = computed({
  get: () => (typeof route.query.q === 'string' ? route.query.q : ''),
  set: (q: string) => router.replace({ query: { ...route.query, q: q || undefined } }),
})

const filtered = computed(() => filterPackages(packages.value, search.value))
const summary = computed(() => {
  const size = formatBytes(filtered.value.reduce((n, p) => n + p.sizeBytes, 0))
  const of = filtered.value.length !== packages.value.length ? ` of ${packages.value.length}` : ''
  return `${filtered.value.length}${of} packages · ${size}`
})
</script>

<template>
  <AppCard title="Package Catalog">
    <template #subtitle>{{ summary }}</template>
    <template #actions>
      <input v-model="search" type="text" placeholder="Filter packages..." class="search" />
      <AppButton size="small" @click="refresh">Refresh</AppButton>
      <AppButton size="small" :to="{ name: 'install' }">How to install</AppButton>
    </template>

    <div class="table-container">
      <table>
        <thead>
          <tr>
            <th>Package</th>
            <th>Version</th>
            <th>Kernel</th>
            <th>Size</th>
            <th>Built</th>
            <th>Download</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="packages.length === 0">
            <td colspan="6" class="empty">
              <template v-if="autoSync">
                No packages published yet. The upstream sync checks Arch Linux {{ describeInterval(settings?.autoCheckInterval) }} and builds missing
                packages automatically, so manual builds are usually not necessary. You can still
                <RouterLink :to="{ name: 'builds' }">start one</RouterLink> to speed things up.
              </template>
              <template v-else>
                No packages published yet. Automatic sync is disabled, so
                <RouterLink :to="{ name: 'builds' }">start a build</RouterLink> to create the first ones.
              </template>
            </td>
          </tr>
          <tr v-else-if="filtered.length === 0">
            <td colspan="6" class="empty">No packages match "{{ search }}"</td>
          </tr>
          <tr v-for="p in filtered" :key="p.filename">
            <td><strong>{{ p.packageName }}</strong></td>
            <td><code>{{ p.version }}</code></td>
            <td class="kernel">{{ p.kernelVersion || '-' }}</td>
            <td>{{ p.sizeHuman }}</td>
            <td class="built">{{ dates.date(p.modTime) }}</td>
            <td>
              <div class="actions">
                <AppButton size="small" :href="p.downloadUrl" download>Download</AppButton>
                <AppButton v-if="findBuildForPackage(p, builds)" size="small" @click="logViewer.open(findBuildForPackage(p, builds)!.name)">
                  Build log
                </AppButton>
                <AppButton size="small" variant="danger" :disabled="removing.has(p.filename)" @click="onRemove(p)">
                  {{ removing.has(p.filename) ? 'Removing...' : 'Remove' }}
                </AppButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </AppCard>
</template>

<style scoped>
.search {
  width: 220px;
}

.kernel {
  color: var(--accent-blue);
}

.actions {
  display: flex;
  gap: 6px;
  white-space: nowrap;
}

.built {
  color: var(--text-secondary);
}
</style>
