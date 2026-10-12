<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useBuilds } from '../../composables/useBuilds'
import { useLogViewer } from '../../composables/useLogViewer'
import { usePackages } from '../../composables/usePackages'
import { useSettings } from '../../composables/useSettings'
import { describeInterval, formatBytes } from '../../utils/format'
import { filterPackages, findBuildForPackage } from '../../utils/packages'
import AppButton from '../ui/AppButton.vue'
import AppCard from '../ui/AppCard.vue'

const { packages, refresh } = usePackages()
const { builds } = useBuilds()
const logViewer = useLogViewer()
// Build rows link here with ?q=<kernel> to show the packages they produced.
const initialQuery = useRoute().query.q
const { settings } = useSettings()
// Unknown (settings not loaded yet) counts as enabled, the default.
const autoSync = computed(() => {
  const interval = settings.value?.autoCheckInterval
  return interval === undefined || !['', '0', '0s'].includes(interval)
})
const search = ref(typeof initialQuery === 'string' ? initialQuery : '')

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
            <td class="built">{{ new Date(p.modTime).toLocaleDateString() }}</td>
            <td>
              <div class="actions">
                <AppButton size="small" :href="p.downloadUrl" download>Download</AppButton>
                <AppButton v-if="findBuildForPackage(p, builds)" size="small" @click="logViewer.open(findBuildForPackage(p, builds)!.name)">
                  Build log
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
