<script setup lang="ts">
import { useLogViewer } from '../../composables/useLogViewer'
import { useNow } from '../../composables/useNow'
import type { Insight } from '../../utils/analytics'
import { formatAgo, formatDuration } from '../../utils/format'
import { buildTarget, retryRoute, variantLabel } from '../../utils/jobs'
import AppButton from '../ui/AppButton.vue'

defineProps<{ insights: Insight[]; loaded: boolean }>()

const now = useNow()
const logViewer = useLogViewer()

const ended = (i: Insight) => i.job.endTime ?? i.job.startTime
</script>

<template>
  <section class="insights" aria-label="Needs attention">
    <div v-if="!loaded" class="insight">
      <div class="text muted">Loading builds&hellip;</div>
    </div>
    <div v-else-if="insights.length === 0" class="insight ok">
      <span class="icon">&#10003;</span>
      <div class="text">
        <strong>All good</strong>
        <span class="muted">No failing or unusually slow builds.</span>
      </div>
    </div>
    <div v-for="i in insights" :key="i.kind + i.variant" class="insight" :class="i.severity">
      <span class="icon">{{ i.severity === 'error' ? '!' : i.severity === 'warning' ? '~' : '&#9654;' }}</span>
      <div class="text">
        <template v-if="i.kind === 'failing'">
          <strong>
            {{ variantLabel(i.variant) }}:
            {{ i.count === 1 ? 'the latest build failed' : `the last ${i.count} builds failed` }}
          </strong>
          <span class="muted">
            {{ buildTarget(i.job) }}, {{ formatAgo(ended(i), now) }}.
            {{ i.lastSuccess ? `Last success ${formatAgo(i.lastSuccess.endTime, now)}.` : 'No successful build on record.' }}
          </span>
        </template>
        <template v-else-if="i.kind === 'slow'">
          <strong>{{ variantLabel(i.variant) }}: the latest build took {{ i.percent }}% longer than usual</strong>
          <span class="muted">{{ formatDuration(i.job.durationSec) }} instead of about {{ formatDuration(i.medianSec) }}, {{ formatAgo(ended(i), now) }}.</span>
        </template>
        <template v-else>
          <strong>{{ variantLabel(i.variant) }}: build running</strong>
          <span class="muted">{{ buildTarget(i.job) }}, started {{ formatAgo(i.job.startTime, now) }}.</span>
        </template>
      </div>
      <div class="actions">
        <AppButton size="small" @click="logViewer.open(i.job.name)">{{ i.kind === 'building' ? 'Watch log' : 'View log' }}</AppButton>
        <AppButton v-if="i.kind === 'failing'" size="small" :to="retryRoute(i.job)">Retry build</AppButton>
      </div>
    </div>
  </section>
</template>

<style scoped>
.insights {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.insight {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border: 1px solid var(--border-color);
  border-left-width: 4px;
  border-radius: 8px;
  background: var(--bg-secondary);
}

.insight.ok {
  border-left-color: var(--status-success);
}

.insight.error {
  border-left-color: var(--status-error);
}

.insight.warning {
  border-left-color: var(--status-warning);
}

.insight.info {
  border-left-color: var(--accent-blue);
}

.icon {
  width: 22px;
  height: 22px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border-radius: 50%;
  font-size: 0.75rem;
  font-weight: 700;
  background: var(--bg-tertiary);
}

.ok .icon {
  color: var(--status-success);
}

.error .icon {
  color: var(--status-error);
}

.warning .icon {
  color: var(--status-warning);
}

.info .icon {
  color: var(--accent-blue);
}

.text {
  flex: 1;
  display: flex;
  flex-direction: column;
  font-size: 0.875rem;
}

.text .muted {
  font-size: 0.8125rem;
}

.actions {
  display: flex;
  gap: 8px;
}
</style>
