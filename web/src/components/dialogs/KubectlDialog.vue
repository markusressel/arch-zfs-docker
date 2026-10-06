<script setup lang="ts">
import { computed } from 'vue'
import { useSettings } from '../../composables/useSettings'
import type { JobSummary } from '../../types/api'
import { kubectlCommands } from '../../utils/kubectl'
import AppButton from '../ui/AppButton.vue'
import AppModal from '../ui/AppModal.vue'
import CopyButton from '../ui/CopyButton.vue'

const props = defineProps<{ job: JobSummary }>()
const emit = defineEmits<{ close: [] }>()

const { settings } = useSettings()
const commands = computed(() => kubectlCommands(props.job.namespace || settings.value?.namespace || 'default', props.job.name))
</script>

<template>
  <AppModal :title="`kubectl: ${job.name}`" @close="emit('close')">
    <div class="body">
      <div v-for="c in commands" :key="c.label" class="command">
        <div class="label">{{ c.label }}</div>
        <div class="line">
          <code>{{ c.command }}</code>
          <CopyButton :text="c.command" />
        </div>
      </div>
      <a v-if="settings?.dashboardUrl" :href="settings.dashboardUrl" target="_blank" rel="noopener" class="dashboard">Open Kubernetes dashboard &rarr;</a>
    </div>
    <template #footer><AppButton @click="emit('close')">Close</AppButton></template>
  </AppModal>
</template>

<style scoped>
.body {
  padding: 4px 16px 16px;
  overflow-y: auto;
}

.label {
  margin-top: 12px;
  color: var(--text-secondary);
  font-size: 0.75rem;
}

.line {
  display: flex;
  gap: 8px;
  align-items: stretch;
  margin-top: 8px;
}

code {
  flex: 1;
  padding: 6px 10px;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 0.75rem;
  overflow-x: auto;
  white-space: nowrap;
}

.dashboard {
  display: inline-block;
  margin-top: 16px;
  color: var(--accent-blue);
  font-size: 0.875rem;
}
</style>
