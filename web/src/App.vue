<script setup lang="ts">
import { onMounted } from 'vue'
import DialogHost from './components/dialogs/DialogHost.vue'
import AppHeader from './components/layout/AppHeader.vue'
import AppTabs, { type Tab } from './components/layout/AppTabs.vue'
import LogModal from './components/logs/LogModal.vue'
import { useBuilds } from './composables/useBuilds'
import { useLogViewer } from './composables/useLogViewer'
import { usePackages } from './composables/usePackages'
import { usePolling } from './composables/usePolling'
import { useSettings } from './composables/useSettings'
import { useUpstream } from './composables/useUpstream'
import { routes } from './router'

const POLL_INTERVAL_MS = 8000

const tabs: Tab[] = routes.map((r) => ({ name: r.name, label: r.meta.label }))

const { load: loadSettings } = useSettings()
const { refresh: refreshBuilds } = useBuilds()
const { refresh: refreshUpstream } = useUpstream()
const { refresh: refreshPackages } = usePackages()
const logViewer = useLogViewer()

function refreshAll() {
  refreshBuilds()
  refreshUpstream()
  refreshPackages()
}

onMounted(() => {
  loadSettings()
  refreshAll()
})
usePolling(() => {
  refreshBuilds()
  refreshUpstream()
}, POLL_INTERVAL_MS)

function closeLogs() {
  logViewer.close()
  // A finished build changes the job list and the package catalog.
  setTimeout(refreshAll, 100)
}
</script>

<template>
  <div class="container">
    <AppHeader />
    <AppTabs :tabs="tabs" />
    <RouterView />
    <LogModal v-if="logViewer.jobName.value" :job-name="logViewer.jobName.value" @close="closeLogs" />
    <DialogHost />
  </div>
</template>

<style scoped>
.container {
  max-width: 1200px;
  margin: 0 auto;
}
</style>
