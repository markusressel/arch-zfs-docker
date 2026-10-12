<script setup lang="ts">
import { computed } from 'vue'
import { useLogViewer } from '../../composables/useLogViewer'
import { useNow } from '../../composables/useNow'
import type { JobSummary } from '../../types/api'
import { formatAgo, formatDuration } from '../../utils/format'
import { buildTarget } from '../../utils/jobs'
import AppCard from '../ui/AppCard.vue'
import BarChart, { type Bar } from '../ui/BarChart.vue'

const MAX_RUNS = 40

const props = defineProps<{
  /** Finished builds matching the filters, oldest first. */
  runs: JobSummary[]
  medianSec: number
}>()

const now = useNow()
const logViewer = useLogViewer()

const shown = computed(() => props.runs.filter((b) => b.durationSec > 0).slice(-MAX_RUNS))
const bars = computed<Bar[]>(() =>
  shown.value.map((b) => ({
    key: b.name,
    value: b.durationSec,
    title: `${buildTarget(b)} — ${b.status}, ${formatDuration(b.durationSec)}, ${formatAgo(b.endTime ?? b.startTime, now.value)}. Click to open the log.`,
    tone: b.status === 'Failed' ? 'danger' : 'primary',
  })),
)
const reference = computed(() => (props.medianSec ? { value: props.medianSec, label: `median ${formatDuration(props.medianSec)}` } : undefined))
</script>

<template>
  <AppCard title="Build duration per run">
    <template #subtitle>
      Latest {{ shown.length }} matching builds, oldest to newest. Click a bar to open its log.
    </template>
    <p v-if="shown.length === 0" class="muted empty">No finished builds match the filters.</p>
    <template v-else>
      <BarChart :bars="bars" clickable :reference="reference" @select="logViewer.open" />
      <div class="legend">
        <span><i class="ok" />Succeeded</span>
        <span><i class="fail" />Failed</span>
        <span v-if="reference"><i class="median" />Median of successful builds</span>
      </div>
    </template>
  </AppCard>
</template>

<style scoped>
.empty {
  margin-top: 16px;
}

.legend {
  display: flex;
  gap: 16px;
  margin-top: 12px;
  font-size: 0.75rem;
  color: var(--text-secondary);
}

i {
  display: inline-block;
  width: 10px;
  height: 10px;
  margin-right: 6px;
  border-radius: 2px;
  vertical-align: -1px;
}

.ok {
  background: var(--accent-blue);
}

.fail {
  background: var(--status-error);
}

.median {
  height: 0;
  border-top: 1px dashed var(--text-muted);
  border-radius: 0;
  vertical-align: middle;
}
</style>
