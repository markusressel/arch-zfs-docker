<script setup lang="ts">
import { computed } from 'vue'
import type { JobSummary } from '../../types/api'
import { formatDuration } from '../../utils/format'
import AppCard from '../ui/AppCard.vue'
import BarChart, { type Bar } from '../ui/BarChart.vue'

const MAX_RUNS = 30

const props = defineProps<{ finished: JobSummary[] }>()

const runs = computed(() => props.finished.slice(-MAX_RUNS))
const bars = computed<Bar[]>(() =>
  runs.value.map((b) => ({
    key: b.name,
    value: b.durationSec,
    title: `${b.name} — ${b.status}, ${formatDuration(b.durationSec)}`,
    tone: b.status === 'Failed' ? 'danger' : 'primary',
  })),
)
const longest = computed(() => Math.max(0, ...runs.value.map((b) => b.durationSec)))
</script>

<template>
  <AppCard title="Build duration per run">
    <template #subtitle>Last {{ MAX_RUNS }} finished jobs, oldest to newest. Hover a bar for details.</template>
    <p v-if="runs.length === 0" class="muted empty">No finished jobs yet.</p>
    <template v-else>
      <BarChart :bars="bars" />
      <div class="legend">
        <span><i class="ok" />Succeeded</span>
        <span><i class="fail" />Failed</span>
        <span>Tallest bar: {{ formatDuration(longest) }}</span>
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
}

.ok {
  background: var(--accent-blue);
}

.fail {
  background: var(--status-error);
}
</style>
