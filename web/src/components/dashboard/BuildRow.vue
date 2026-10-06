<script setup lang="ts">
import { computed } from 'vue'
import { useNow } from '../../composables/useNow'
import type { JobStatus, JobSummary } from '../../types/api'
import { estimateDuration, etaText, progressPercent } from '../../utils/estimate'
import { formatDuration } from '../../utils/format'
import AppButton from '../ui/AppButton.vue'
import ProgressBar from '../ui/ProgressBar.vue'
import StatusPill, { type PillTone } from '../ui/StatusPill.vue'

const props = defineProps<{
  job: JobSummary
  /** All known jobs (newest first), used to estimate the remaining time. */
  history: JobSummary[]
}>()

const emit = defineEmits<{ logs: []; kubectl: []; delete: [] }>()

const tones: Record<JobStatus, PillTone> = { Succeeded: 'success', Running: 'running', Failed: 'error', Pending: 'warning' }

const now = useNow()
const running = computed(() => props.job.status === 'Running' && !!props.job.startTime)
const finished = computed(() => props.job.status === 'Succeeded' || props.job.status === 'Failed')

const elapsed = computed(() =>
  running.value ? Math.max(0, Math.floor((now.value - new Date(props.job.startTime!).getTime()) / 1000)) : props.job.durationSec,
)
const estimate = computed(() => (running.value ? estimateDuration(props.history, props.job.variant) : 0))
const startedAt = computed(() => (props.job.startTime ? new Date(props.job.startTime).toLocaleTimeString() : '-'))
const kernelLabel = computed(() => {
  const { variant, kernelVersion, zfsVersion } = props.job
  if (!variant && !kernelVersion && !zfsVersion) return ''
  return [variant ? 'lts' : 'linux', kernelVersion, zfsVersion && `zfs ${zfsVersion}`].filter(Boolean).join(' ')
})
</script>

<template>
  <tr>
    <td>
      <code>{{ job.name }}</code>
      <div v-if="kernelLabel" class="muted small">{{ kernelLabel }}</div>
    </td>
    <td><StatusPill :tone="tones[job.status]">{{ job.status }}</StatusPill></td>
    <td class="started">{{ startedAt }}</td>
    <td>
      <span class="duration">{{ running || job.durationSec > 0 ? formatDuration(elapsed) : '-' }}</span>
      <template v-if="running && estimate">
        <span class="eta">{{ etaText(elapsed, estimate) }}</span>
        <ProgressBar :percent="progressPercent(elapsed, estimate)" />
      </template>
    </td>
    <td>
      <div class="actions">
        <AppButton size="small" @click="emit('logs')">Logs</AppButton>
        <AppButton size="small" @click="emit('kubectl')">kubectl</AppButton>
        <AppButton v-if="finished" size="small" @click="emit('delete')">Delete</AppButton>
      </div>
    </td>
  </tr>
</template>

<style scoped>
.started {
  color: var(--text-secondary);
}

.duration {
  display: inline-block;
  min-width: 9ch;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.eta {
  display: block;
  font-size: 0.6875rem;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.actions {
  display: flex;
  gap: 6px;
  white-space: nowrap;
}
</style>
