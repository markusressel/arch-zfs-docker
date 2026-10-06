<script setup lang="ts">
import { computed, ref } from 'vue'
import { usePackages } from '../../composables/usePackages'
import { formatBytes } from '../../utils/format'
import { filterPackages } from '../../utils/packages'
import AppButton from '../ui/AppButton.vue'
import AppCard from '../ui/AppCard.vue'

const { packages, refresh } = usePackages()
const search = ref('')

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
          <tr v-if="filtered.length === 0">
            <td colspan="6" class="empty">No packages found</td>
          </tr>
          <tr v-for="p in filtered" :key="p.filename">
            <td><strong>{{ p.packageName }}</strong></td>
            <td><code>{{ p.version }}</code></td>
            <td class="kernel">{{ p.kernelVersion || '-' }}</td>
            <td>{{ p.sizeHuman }}</td>
            <td class="built">{{ new Date(p.modTime).toLocaleDateString() }}</td>
            <td><AppButton size="small" :href="p.downloadUrl" download>Download</AppButton></td>
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

.built {
  color: var(--text-secondary);
}
</style>
