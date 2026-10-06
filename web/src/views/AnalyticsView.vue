<script setup lang="ts">
import { computed } from 'vue'
import DurationHistogram from '../components/analytics/DurationHistogram.vue'
import KpiCards from '../components/analytics/KpiCards.vue'
import RunsChart from '../components/analytics/RunsChart.vue'
import { useBuilds } from '../composables/useBuilds'
import { computeAnalytics } from '../utils/analytics'

const { builds } = useBuilds()
const analytics = computed(() => computeAnalytics(builds.value))
</script>

<template>
  <div class="analytics">
    <KpiCards :analytics="analytics" />
    <RunsChart :finished="analytics.finished" />
    <DurationHistogram :finished="analytics.finished" />
    <p class="muted small">
      Based on the build jobs currently present in the cluster &mdash; clearing finished jobs also removes them from these statistics.
    </p>
  </div>
</template>

<style scoped>
.analytics {
  display: flex;
  flex-direction: column;
  gap: 24px;
}
</style>
