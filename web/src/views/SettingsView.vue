<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import DisplaySettings from '../components/settings/DisplaySettings.vue'
import AppButton from '../components/ui/AppButton.vue'
import AppCard from '../components/ui/AppCard.vue'
import FormField from '../components/ui/FormField.vue'
import { useSettings } from '../composables/useSettings'

const INTERVALS = [
  ['1h', 'Every 1 hour'],
  ['3h', 'Every 3 hours'],
  ['6h', 'Every 6 hours (Default)'],
  ['12h', 'Every 12 hours'],
  ['24h', 'Every 24 hours (Daily)'],
  ['0', 'Disabled (Manual trigger only)'],
] as const

const { settings, load, save } = useSettings()

const form = reactive({ autoCheckInterval: '6h', buildNode: '' })
const saving = ref(false)
const toast = ref<{ ok: boolean; text: string } | null>(null)
let toastTimer: ReturnType<typeof setTimeout> | undefined

function reset() {
  if (!settings.value) return
  form.autoCheckInterval = settings.value.autoCheckInterval || '6h'
  form.buildNode = settings.value.buildNode || ''
}

watch(settings, reset, { immediate: true })

function showToast(ok: boolean, text: string) {
  toast.value = { ok, text }
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => (toast.value = null), 4000)
}

async function onSubmit() {
  saving.value = true
  try {
    await save({
      autoCheckInterval: form.autoCheckInterval,
      buildNode: form.buildNode.trim(),
    })
    showToast(true, 'Settings saved successfully!')
  } catch (err) {
    showToast(false, `Failed to save settings: ${err instanceof Error ? err.message : err}`)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="settings">
    <AppCard title="Dynamic Service Settings">
      <p class="muted intro">Adjust upstream polling schedules and Kubernetes builder node assignment without restarting the container.</p>

      <v-form class="d-flex flex-column ga-4" @submit.prevent="onSubmit">
        <FormField hint="How frequently the background poller checks Arch Linux official repos.">
          <v-select
            id="setting-interval"
            v-model="form.autoCheckInterval"
            :items="INTERVALS.map(([value, title]) => ({ value, title }))"
            label="Upstream Auto-Check Interval"
          />
        </FormField>

        <FormField>
          <v-text-field id="setting-runner" label="Build Runner" readonly :model-value="settings?.runner ?? ''" />
          <template #hint>
            Where builds run: <code>kubernetes</code> or the local <code>docker</code> engine (used when no cluster is available). Configured via
            <code>BUILD_RUNNER</code> (<code>auto</code>, <code>kubernetes</code>, <code>docker</code>).
          </template>
        </FormField>

        <FormField>
          <v-text-field id="setting-build-node" v-model="form.buildNode" label="Target Build Node (nodeSelector)" placeholder="e.g. worker-node-1" />
          <template #hint>Pins build jobs to a specific Kubernetes node via <code>kubernetes.io/hostname</code>.</template>
        </FormField>

        <FormField>
          <v-text-field id="setting-namespace" label="Kubernetes Namespace" readonly :model-value="settings?.namespace ?? ''" />
          <template #hint>Configured via <code>K8S_NAMESPACE</code> environment variable.</template>
        </FormField>

        <FormField>
          <v-text-field
            id="setting-dashboard"
            label="Kubernetes Dashboard URL"
            readonly
            placeholder="not configured"
            persistent-placeholder
            :model-value="settings?.dashboardUrl ?? ''"
          />
          <template #hint>Configured via <code>K8S_DASHBOARD_URL</code>; adds a link to the build job's kubectl dialog.</template>
        </FormField>

        <FormField>
          <v-text-field id="setting-repo-name" label="Pacman Repository Name" readonly :model-value="settings?.repoName ?? ''" />
          <template #hint>Configured via <code>REPO_NAME</code> environment variable.</template>
        </FormField>

        <div class="d-flex ga-2 mt-2">
          <AppButton type="submit" variant="primary" :disabled="saving">{{ saving ? 'Saving...' : 'Save Settings' }}</AppButton>
          <AppButton @click="load().then(reset)">Reload</AppButton>
        </div>
        <v-alert v-if="toast" :type="toast.ok ? 'success' : 'error'" class="toast">{{ toast.text }}</v-alert>
      </v-form>
    </AppCard>
    <DisplaySettings />
  </div>
</template>

<style scoped>
.settings {
  display: flex;
  flex-direction: column;
  gap: 24px;
  max-width: 650px;
  margin: 0 auto;
}

.intro {
  margin-bottom: 20px;
  font-size: 0.8125rem;
}

/* Read-only settings come from the environment; dim them like disabled fields. */
.v-text-field:has(input[readonly]) {
  opacity: 0.7;
}
</style>
