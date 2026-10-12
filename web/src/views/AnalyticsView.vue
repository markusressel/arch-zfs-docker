<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DurationHistogram from '../components/analytics/DurationHistogram.vue'
import InsightList from '../components/analytics/InsightList.vue'
import KpiCards from '../components/analytics/KpiCards.vue'
import RunFilters from '../components/analytics/RunFilters.vue'
import RunsChart from '../components/analytics/RunsChart.vue'
import RunsTable from '../components/analytics/RunsTable.vue'
import VariantHealthCard from '../components/analytics/VariantHealthCard.vue'
import { useBuilds } from '../composables/useBuilds'
import { usePackages } from '../composables/usePackages'
import { useUpstream } from '../composables/useUpstream'
import { computeAnalytics, filterFromQuery, filterRuns, filterToQuery, insights, variantHealth, type RunFilter } from '../utils/analytics'
import { KNOWN_VARIANTS, matchesVariant } from '../utils/jobs'

const { builds, loaded } = useBuilds()
const { packages } = usePackages()
const { variants: upstream } = useUpstream()

// Filters live in the URL, so a filtered view can be linked and survives a reload.
const route = useRoute()
const router = useRouter()
const filter = computed<RunFilter>({
  get: () => filterFromQuery(route.query),
  set: (f) => router.replace({ query: filterToQuery(f) }),
})

const health = computed(() => KNOWN_VARIANTS.map((v) => variantHealth(builds.value, v)))
const attention = computed(() => insights(builds.value))
const medianByVariant = computed(() => Object.fromEntries(health.value.map((h) => [h.variant, h.medianSec])))

// KPIs and the histogram follow the kernel filter only; they are what the other filters select from.
const scoped = computed(() => {
  const v = filter.value.variant
  return v === undefined ? builds.value : builds.value.filter((b) => matchesVariant(b, v))
})
const analytics = computed(() => computeAnalytics(scoped.value))
const byStatus = computed(() => filterRuns(analytics.value.finished, { status: filter.value.status }))
const runs = computed(() => filterRuns(analytics.value.finished, filter.value))

const toggleVariant = (variant: string) =>
  (filter.value = { ...filter.value, variant: filter.value.variant === variant ? undefined : variant })
</script>

<template>
  <div class="analytics">
    <InsightList :insights="attention" :loaded="loaded" />

    <div class="health">
      <VariantHealthCard
        v-for="h in health"
        :key="h.variant"
        :health="h"
        :upstream="upstream.find((u) => (u.variant || '') === h.variant)"
        :packages="packages"
        :selected="filter.variant === h.variant"
        @select="toggleVariant(h.variant)"
      />
    </div>

    <RunFilters v-model="filter" />

    <KpiCards :analytics="analytics" :status="filter.status" @status="(status) => (filter = { ...filter, status })" />
    <RunsChart :runs="runs" :median-sec="analytics.medianBuildSec" />
    <DurationHistogram :runs="byStatus" :selected="filter.bucket" @select="(bucket) => (filter = { ...filter, bucket })" />
    <RunsTable :runs="runs" :median-by-variant="medianByVariant" :packages="packages" />

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

.health {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 16px;
}
</style>
