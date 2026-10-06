<script setup lang="ts">
import { computed } from 'vue'
import type { JobSummary } from '../../types/api'
import { durationBuckets } from '../../utils/analytics'
import AppCard from '../ui/AppCard.vue'
import BarChart, { type Bar } from '../ui/BarChart.vue'

const props = defineProps<{ finished: JobSummary[] }>()

const bars = computed<Bar[]>(() =>
  durationBuckets(props.finished).map((b) => ({
    key: b.label,
    value: b.count,
    label: b.label,
    showValue: true,
    title: `${b.count} job${b.count === 1 ? '' : 's'}`,
  })),
)
</script>

<template>
  <AppCard title="Duration distribution">
    <template #subtitle>Number of finished jobs per duration bucket.</template>
    <p v-if="finished.length === 0" class="muted empty">No finished jobs yet.</p>
    <BarChart v-else :bars="bars" max-bar-width="90px" :fill="85" />
  </AppCard>
</template>

<style scoped>
.empty {
  margin-top: 16px;
}
</style>
