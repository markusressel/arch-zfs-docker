<script setup lang="ts">
import { computed } from 'vue'
import { useNow } from '../../composables/useNow'
import type { JobSummary, VariantStatus } from '../../types/api'
import { buildEta } from '../../utils/estimate'
import { formatDuration } from '../../utils/format'
import ProgressBar from '../ui/ProgressBar.vue'
import StatusPill from '../ui/StatusPill.vue'

const props = defineProps<{
  status: VariantStatus
  /** Active build job for this kernel variant, if any. */
  job?: JobSummary
  /** All known jobs (newest first), used to estimate the remaining build time. */
  history: JobSummary[]
}>()

const now = useNow()
const building = computed(() => !!props.job)
const eta = computed(() => (props.job ? buildEta(props.job, props.history, now.value) : null))

const etaLabel = computed(() => {
  const e = eta.value
  if (!e || props.status.upToDate) return ''
  if (!e.estimateSec) return `Build running for ${formatDuration(e.elapsedSec)} (no estimate yet)`
  if (e.overdue) return 'Build is taking longer than usual'
  const at = new Date(e.finishAt!).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  const min = Math.max(1, Math.round(e.remainingSec / 60))
  const left = min >= 90 ? `${Math.round(min / 60)} h` : `${min} min`
  return `in about ${left} (around ${at})`
})

const pill = computed(() => {
  if (props.status.upToDate) return { tone: 'success', text: 'Up to date', link: undefined } as const
  if (building.value) {
    return { tone: 'running', text: 'Building', link: { to: { name: 'builds' }, hint: 'Show running build jobs' } } as const
  }
  return {
    tone: 'warning',
    text: 'Build needed',
    link: {
      to: { name: 'builds', query: { kernel: props.status.upstreamKernel, variant: props.status.variant } },
      hint: `Start a build for ${props.status.upstreamKernel}`,
    },
  } as const
})
</script>

<template>
  <div class="item" :class="status.upToDate ? 'ok' : 'behind'">
    <div class="head">
      <strong>{{ status.kernelPkg }}</strong>
      <RouterLink v-if="pill.link" :to="pill.link.to" :title="pill.link.hint" class="pill-link">
        <StatusPill :tone="pill.tone">{{ pill.text }} &rarr;</StatusPill>
      </RouterLink>
      <StatusPill v-else :tone="pill.tone">{{ pill.text }}</StatusPill>
    </div>
    <div class="row"><span>Arch upstream</span><code>{{ status.upstreamKernel }}</code></div>
    <div class="row"><span>Our repository</span><code>{{ status.localKernel || 'none built' }}</code></div>
    <div v-if="etaLabel" class="row eta">
      <span>Package available</span><strong>{{ etaLabel }}</strong>
    </div>
    <ProgressBar v-if="etaLabel && eta?.estimateSec && !eta.overdue" :percent="eta.percent" />
  </div>
</template>

<style scoped>
.item {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-left: 3px solid var(--text-muted);
  border-radius: 6px;
  padding: 14px 16px;
}

.ok {
  border-left-color: var(--status-success);
}

.behind {
  border-left-color: var(--status-warning);
}

.head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.pill-link {
  border-radius: 9999px;
  text-decoration: none;
}

.pill-link:hover :deep(.pill),
.pill-link:focus-visible :deep(.pill) {
  filter: brightness(1.3);
  text-decoration: underline;
}

.row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  font-size: 0.8125rem;
  padding: 3px 0;
}

.eta strong {
  color: var(--accent-blue);
  font-weight: 600;
  text-align: right;
}

.row span {
  color: var(--text-secondary);
}
</style>
