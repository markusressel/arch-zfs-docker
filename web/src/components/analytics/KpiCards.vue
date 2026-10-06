<script setup lang="ts">
import { computed } from 'vue'
import type { Analytics } from '../../utils/analytics'
import { formatDuration } from '../../utils/format'
import AppCard from '../ui/AppCard.vue'

const props = defineProps<{ analytics: Analytics }>()

const kpis = computed(() => {
  const a = props.analytics
  return [
    { title: 'Finished jobs', value: String(a.finishedCount) },
    { title: 'Success rate', value: a.successRate === null ? '-' : `${a.successRate}%` },
    { title: 'Median build time', value: a.medianBuildSec ? formatDuration(a.medianBuildSec) : '-' },
    { title: 'Last success', value: a.lastSuccessAt ? new Date(a.lastSuccessAt).toLocaleDateString() : '-' },
  ]
})
</script>

<template>
  <div class="grid">
    <AppCard v-for="k in kpis" :key="k.title">
      <div class="title">{{ k.title }}</div>
      <div class="value">{{ k.value }}</div>
    </AppCard>
  </div>
</template>

<style scoped>
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 16px;
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
</style>
