<script setup lang="ts">
import { computed, onMounted } from 'vue'
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

const { load: loadSettings } = useSettings()
const { builds, isActive, refresh: refreshBuilds } = useBuilds()
const { refresh: refreshUpstream } = useUpstream()
const { refresh: refreshPackages } = usePackages()
const logViewer = useLogViewer()

const activeBuilds = computed(() => builds.value.filter(isActive).length)
const tabs = computed<Tab[]>(() =>
  routes.map((r) => ({ name: r.name, label: r.meta.label, badge: r.name === 'builds' ? activeBuilds.value : undefined })),
)

function refreshAll() {
  refreshBuilds()
  refreshUpstream()
  refreshPackages()
}

onMounted(() => {
  loadSettings()
  refreshAll()
})
// The package catalog is polled too, so packages published by a finished job show up on their own.
usePolling(refreshAll, POLL_INTERVAL_MS)

function closeLogs() {
  logViewer.close()
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
