<script setup lang="ts">
import { computed } from 'vue'
import { useLogViewer } from '../../composables/useLogViewer'
import { useNow } from '../../composables/useNow'
import type { Analytics, RunStatus } from '../../utils/analytics'
import { formatAgo, formatDuration } from '../../utils/format'
import { buildTarget } from '../../utils/jobs'

const props = defineProps<{ analytics: Analytics; status?: RunStatus }>()
const emit = defineEmits<{ status: [status: RunStatus | undefined] }>()

const now = useNow()
const logViewer = useLogViewer()

type Tone = 'ok' | 'warn' | 'bad' | undefined

interface Kpi {
  title: string
  value: string
  detail: string
  tone?: Tone
  hint?: string
  selected?: boolean
  action?: () => void
}

const toggle = (s: RunStatus) => () => emit('status', props.status === s ? undefined : s)

const kpis = computed<Kpi[]>(() => {
  const a = props.analytics
  const rate = a.successRate
  const last = a.lastSuccess
  return [
    {
      title: 'Success rate',
      value: rate === null ? '-' : `${rate}%`,
      detail: a.finishedCount ? `${a.finishedCount - a.failedCount} of ${a.finishedCount} builds succeeded` : 'no finished builds',
      tone: rate === null ? undefined : rate >= 80 ? 'ok' : rate >= 50 ? 'warn' : 'bad',
      hint: 'Show succeeded builds',
      selected: props.status === 'Succeeded',
      action: a.finishedCount ? toggle('Succeeded') : undefined,
    },
    {
      title: 'Failed builds',
      value: String(a.failedCount),
      detail: a.failedCount ? 'click to list them' : 'nothing to look at',
      tone: a.failedCount ? 'bad' : 'ok',
      hint: 'Show failed builds',
      selected: props.status === 'Failed',
      action: a.failedCount ? toggle('Failed') : undefined,
    },
    {
      title: 'Typical build time',
      value: a.medianBuildSec ? formatDuration(a.medianBuildSec) : '-',
      detail: 'median of successful builds',
    },
    {
      title: 'Last success',
      value: last ? formatAgo(last.endTime, now.value) : '-',
      detail: last ? buildTarget(last) : 'none on record',
      tone: last ? undefined : 'bad',
      hint: 'Open the log of this build',
      action: last ? () => logViewer.open(last.name) : undefined,
    },
  ]
})
</script>

<template>
  <div class="grid">
    <component
      :is="k.action ? 'button' : 'div'"
      v-for="k in kpis"
      :key="k.title"
      :type="k.action ? 'button' : undefined"
      class="kpi"
      :class="[k.tone, { clickable: !!k.action, selected: k.selected }]"
      :title="k.action ? k.hint : undefined"
      :aria-pressed="k.action && k.selected !== undefined ? k.selected : undefined"
      @click="k.action?.()"
    >
      <div class="title">{{ k.title }}</div>
      <div class="value">{{ k.value }}</div>
      <div class="detail">{{ k.detail }}</div>
    </component>
  </div>
</template>

<style scoped>
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 16px;
}

.kpi {
  padding: 16px 18px;
  text-align: left;
  font: inherit;
  color: inherit;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
}

.kpi.clickable {
  cursor: pointer;
  transition: border-color 0.15s;
}

.kpi.clickable:hover,
.kpi.clickable:focus-visible {
  border-color: var(--text-muted);
}

.kpi.selected {
  border-color: var(--accent-blue);
  background: rgba(88, 166, 255, 0.06);
}

.title {
  margin-bottom: 8px;
  font-size: 0.8125rem;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.value {
  font-size: 1.5rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.ok .value {
  color: var(--status-success);
}

.warn .value {
  color: var(--status-warning);
}

.bad .value {
  color: var(--status-error);
}

.detail {
  margin-top: 4px;
  font-size: 0.75rem;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
