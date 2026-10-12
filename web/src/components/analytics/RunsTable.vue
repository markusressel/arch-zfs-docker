<script setup lang="ts">
import { computed, ref } from 'vue'
import { useLogViewer } from '../../composables/useLogViewer'
import { useNow } from '../../composables/useNow'
import type { JobSummary, PackageInfo } from '../../types/api'
import { durationDelta } from '../../utils/analytics'
import { formatAgo, formatDuration } from '../../utils/format'
import { buildTarget, retryRoute } from '../../utils/jobs'
import { packagesForJob } from '../../utils/packages'
import AppButton from '../ui/AppButton.vue'
import AppCard from '../ui/AppCard.vue'
import StatusPill from '../ui/StatusPill.vue'

const PAGE = 15

const props = defineProps<{
  /** Finished builds matching the filters, oldest first. */
  runs: JobSummary[]
  /** Median build time per kernel variant, to compare each build with its own kind. */
  medianByVariant: Record<string, number>
  packages: PackageInfo[]
}>()

const now = useNow()
const logViewer = useLogViewer()
const limit = ref(PAGE)

const newestFirst = computed(() => [...props.runs].reverse())
const shown = computed(() => newestFirst.value.slice(0, limit.value))

const packagesKernel = (job: JobSummary) => packagesForJob(job, props.packages)[0]?.kernelVersion
const delta = (job: JobSummary) => durationDelta(job, props.medianByVariant[job.variant || ''] ?? 0)
</script>

<template>
  <AppCard title="Builds">
    <template #subtitle>{{ runs.length }} matching finished build{{ runs.length === 1 ? '' : 's' }}, newest first. Click a row to open its log.</template>
    <template #actions>
      <AppButton size="small" :to="{ name: 'builds' }">Build history</AppButton>
    </template>
    <div class="table-container">
      <table>
        <thead>
          <tr>
            <th>Result</th>
            <th>Target</th>
            <th>Finished</th>
            <th>Duration</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="runs.length === 0">
            <td colspan="5" class="empty">No finished builds match the filters</td>
          </tr>
          <tr v-for="job in shown" :key="job.name" class="row" tabindex="0" @click="logViewer.open(job.name)" @keydown.enter="logViewer.open(job.name)">
            <td><StatusPill :tone="job.status === 'Failed' ? 'error' : 'success'">{{ job.status }}</StatusPill></td>
            <td>
              {{ buildTarget(job) }}
              <div class="muted small"><code>{{ job.name }}</code></div>
            </td>
            <td class="muted">{{ formatAgo(job.endTime ?? job.startTime, now) }}</td>
            <td>
              <span class="duration">{{ job.durationSec > 0 ? formatDuration(job.durationSec) : '-' }}</span>
              <span
                v-if="delta(job) !== null && Math.abs(delta(job)!) >= 10"
                class="delta"
                :class="delta(job)! > 0 ? 'slower' : 'faster'"
                title="Compared with the median of successful builds for this kernel"
              >
                {{ delta(job)! > 0 ? '+' : '' }}{{ delta(job) }}%
              </span>
            </td>
            <td>
              <div class="actions" @click.stop @keydown.enter.stop>
                <AppButton size="small" @click="logViewer.open(job.name)">Log</AppButton>
                <AppButton v-if="packagesKernel(job)" size="small" :to="{ name: 'dashboard', query: { q: packagesKernel(job) } }">Packages</AppButton>
                <AppButton v-if="job.status === 'Failed'" size="small" :to="retryRoute(job)">Retry</AppButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-if="runs.length > shown.length" class="more">
      <AppButton size="small" @click="limit += PAGE">Show more ({{ runs.length - shown.length }} left)</AppButton>
    </div>
  </AppCard>
</template>

<style scoped>
.row {
  cursor: pointer;
}

.row:focus-visible td {
  background: rgba(88, 166, 255, 0.06);
}

.duration {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}

.delta {
  margin-left: 8px;
  font-size: 0.75rem;
  font-variant-numeric: tabular-nums;
}

.slower {
  color: var(--status-warning);
}

.faster {
  color: var(--status-success);
}

.actions {
  display: flex;
  gap: 8px;
}

.more {
  margin-top: 12px;
  text-align: center;
}
</style>
