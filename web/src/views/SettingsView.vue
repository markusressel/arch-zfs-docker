<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
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
    await save({ autoCheckInterval: form.autoCheckInterval, buildNode: form.buildNode.trim() })
    showToast(true, 'Settings saved successfully!')
  } catch (err) {
    showToast(false, `Failed to save settings: ${err instanceof Error ? err.message : err}`)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <AppCard title="Dynamic Service Settings" class="settings">
    <p class="muted intro">Adjust upstream polling schedules and Kubernetes builder node assignment without restarting the container.</p>

    <form @submit.prevent="onSubmit">
      <FormField label="Upstream Auto-Check Interval" for="setting-interval" hint="How frequently the background poller checks Arch Linux official repos.">
        <select id="setting-interval" v-model="form.autoCheckInterval">
          <option v-for="[value, label] in INTERVALS" :key="value" :value="value">{{ label }}</option>
        </select>
      </FormField>

      <FormField label="Target Build Node (nodeSelector)" for="setting-build-node">
        <input id="setting-build-node" v-model="form.buildNode" type="text" placeholder="e.g. worker-node-1" />
        <template #hint>Pins build jobs to a specific Kubernetes node via <code>kubernetes.io/hostname</code>.</template>
      </FormField>

      <FormField label="Kubernetes Namespace" for="setting-namespace">
        <input id="setting-namespace" type="text" readonly :value="settings?.namespace ?? ''" />
        <template #hint>Configured via <code>K8S_NAMESPACE</code> environment variable.</template>
      </FormField>

      <FormField label="Kubernetes Dashboard URL" for="setting-dashboard">
        <input id="setting-dashboard" type="text" readonly placeholder="not configured" :value="settings?.dashboardUrl ?? ''" />
        <template #hint>Configured via <code>K8S_DASHBOARD_URL</code>; adds a link to the build job's kubectl dialog.</template>
      </FormField>

      <FormField label="Pacman Repository Name" for="setting-repo-name">
        <input id="setting-repo-name" type="text" readonly :value="settings?.repoName ?? ''" />
        <template #hint>Configured via <code>REPO_NAME</code> environment variable.</template>
      </FormField>

      <div class="buttons">
        <AppButton type="submit" variant="primary" :disabled="saving">{{ saving ? 'Saving...' : 'Save Settings' }}</AppButton>
        <AppButton @click="load().then(reset)">Reload</AppButton>
      </div>
      <p v-if="toast" class="toast" :class="toast.ok ? 'ok' : 'err'">{{ toast.text }}</p>
    </form>
  </AppCard>
</template>

<style scoped>
.settings {
  max-width: 650px;
  margin: 0 auto;
}

.intro {
  margin-bottom: 20px;
  font-size: 0.8125rem;
}

input[readonly] {
  opacity: 0.7;
  cursor: not-allowed;
}

.buttons {
  display: flex;
  gap: 10px;
  margin-top: 24px;
}

.toast {
  margin-top: 14px;
  padding: 10px 14px;
  border-radius: 6px;
  font-size: 0.85rem;
}

.ok {
  background: rgba(63, 185, 80, 0.15);
  border: 1px solid rgba(63, 185, 80, 0.4);
  color: var(--status-success);
}

.err {
  background: rgba(248, 81, 73, 0.15);
  border: 1px solid rgba(248, 81, 73, 0.4);
  color: var(--status-error);
}
</style>
