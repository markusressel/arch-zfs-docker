<script setup lang="ts">
import { computed } from 'vue'
import type { JobSummary } from '../../types/api'
import { durationBuckets } from '../../utils/analytics'
import AppCard from '../ui/AppCard.vue'
import BarChart, { type Bar } from '../ui/BarChart.vue'

const props = defineProps<{ runs: JobSummary[]; selected?: string }>()
const emit = defineEmits<{ select: [bucket: string | undefined] }>()

const buckets = computed(() => durationBuckets(props.runs))
const bars = computed<Bar[]>(() =>
  buckets.value.map((b) => ({
    key: b.label,
    value: b.count,
    label: b.label,
    showValue: true,
    selected: b.label === props.selected,
    title: `${b.count} build${b.count === 1 ? '' : 's'} took ${b.label}. Click to list them.`,
  })),
)
const empty = computed(() => buckets.value.every((b) => b.count === 0))

const select = (bucket: string) => emit('select', bucket === props.selected ? undefined : bucket)
</script>

<template>
  <AppCard title="Duration distribution">
    <template #subtitle>Builds per duration range. Click a range to list its builds.</template>
    <p v-if="empty" class="muted empty">No finished builds match the filters.</p>
    <BarChart v-else :bars="bars" max-bar-width="90px" :fill="85" clickable @select="select" />
  </AppCard>
</template>

<style scoped>
.empty {
  margin-top: 16px;
}
</style>
