<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useTheme } from 'vuetify'
import DialogHost from './components/dialogs/DialogHost.vue'
import AppHeader from './components/layout/AppHeader.vue'
import AppTabs, { type Tab } from './components/layout/AppTabs.vue'
import LogModal from './components/logs/LogModal.vue'
import { useBuilds } from './composables/useBuilds'
import { useLogViewer } from './composables/useLogViewer'
import { usePackages } from './composables/usePackages'
import { usePolling } from './composables/usePolling'
import { useSettings } from './composables/useSettings'
import { useThemeChoice } from './composables/useThemeChoice'
import { useUpstream } from './composables/useUpstream'
import { routes } from './router'

// While a build runs the page follows it closely; otherwise a relaxed rate is enough.
const POLL_ACTIVE_MS = 2000
const POLL_IDLE_MS = 8000

const { load: loadSettings } = useSettings()
const { activeCount: activeBuilds, refresh: refreshBuilds } = useBuilds()
const { refresh: refreshUpstream } = useUpstream()
const { refresh: refreshPackages } = usePackages()
const logViewer = useLogViewer()

const theme = useTheme()
const { choice: themeChoice } = useThemeChoice()
watch(themeChoice, (c) => theme.change(c), { immediate: true })

const tabs = computed<Tab[]>(() => routes.map((r) => ({ name: r.name, label: r.meta.label, badge: r.name === 'builds' ? activeBuilds.value : undefined })))

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
usePolling(refreshAll, () => (activeBuilds.value > 0 ? POLL_ACTIVE_MS : POLL_IDLE_MS))

function closeLogs() {
  logViewer.close()
  setTimeout(refreshAll, 100)
}
</script>

<template>
  <v-app>
    <AppHeader />
    <v-main>
      <v-container class="app-container">
        <AppTabs :tabs="tabs" />
        <RouterView />
      </v-container>
    </v-main>
    <LogModal v-if="logViewer.jobName.value" :job-name="logViewer.jobName.value" @close="closeLogs" />
    <DialogHost />
  </v-app>
</template>

<style scoped>
.app-container {
  max-width: 1200px;
}
</style>
