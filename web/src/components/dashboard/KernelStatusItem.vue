<script setup lang="ts">
import { computed } from 'vue'
import { useLogViewer } from '../../composables/useLogViewer'
import { useNow } from '../../composables/useNow'
import type { JobSummary, VariantStatus } from '../../types/api'
import { buildEta } from '../../utils/estimate'
import { formatDuration } from '../../utils/format'
import CompatNotice from './CompatNotice.vue'
import AppButton from '../ui/AppButton.vue'
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
const logViewer = useLogViewer()

// The newest finished job of this variant, when it failed and nothing is running or has fixed it since.
const failedJob = computed(() => {
  if (props.status.upToDate || props.job) return undefined
  const last = props.history.find((b) => (b.variant || '') === (props.status.variant || '') && b.status !== 'Pending' && b.status !== 'Running')
  return last?.status === 'Failed' ? last : undefined
})
const failedAgo = computed(() => {
  const end = failedJob.value?.endTime ?? failedJob.value?.startTime
  if (!end) return ''
  const min = Math.max(0, Math.round((now.value - new Date(end).getTime()) / 60000))
  return min < 1 ? 'just now' : min < 60 ? `${min} min ago` : `${Math.round(min / 60)} h ago`
})
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
  if (props.status.compat?.state === 'incompatible') {
    return {
      tone: 'error',
      text: 'Waiting for ZFS',
      link: { to: { name: 'builds', query: { kernel: props.status.upstreamKernel, variant: props.status.variant } }, hint: 'Build paused: this kernel is not supported by OpenZFS yet. Click to try a build anyway.' },
    } as const
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
  <div class="item" :class="status.upToDate ? 'ok' : status.compat?.state === 'incompatible' ? 'blocked' : 'behind'">
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
    <CompatNotice v-if="!status.upToDate && status.compat" :compat="status.compat" :kernel="status.upstreamKernel" />
    <div v-if="failedJob && status.compat?.state !== 'incompatible'" class="failed" role="status">
      <strong>The last build failed ({{ failedAgo }}).</strong>
      <p>
        The package for this kernel is still missing. The sync retries on its next check, or start one right away with "Check Now". Open the log to see
        what went wrong.
      </p>
      <AppButton size="small" @click="logViewer.open(failedJob.name)">Show build log</AppButton>
    </div>
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

.blocked {
  border-left-color: var(--status-error);
}

.head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.failed {
  margin-top: 12px;
  padding: 10px 12px;
  border-radius: 6px;
  border: 1px solid rgba(248, 81, 73, 0.4);
  background: rgba(248, 81, 73, 0.08);
  font-size: 0.8125rem;
  line-height: 1.45;
}

.failed p {
  margin: 4px 0 8px;
  color: var(--text-secondary);
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
