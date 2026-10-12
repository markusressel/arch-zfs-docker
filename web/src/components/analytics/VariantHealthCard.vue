<script setup lang="ts">
import { computed } from 'vue'
import { useLogViewer } from '../../composables/useLogViewer'
import { useNow } from '../../composables/useNow'
import type { PackageInfo, VariantStatus } from '../../types/api'
import type { HealthState, VariantHealth } from '../../utils/analytics'
import { formatAgo, formatDuration } from '../../utils/format'
import { buildTarget, variantLabel } from '../../utils/jobs'
import { packagesForJob } from '../../utils/packages'
import AppButton from '../ui/AppButton.vue'
import StatusPill, { type PillTone } from '../ui/StatusPill.vue'

const props = defineProps<{
  health: VariantHealth
  /** Upstream sync state of the variant, if known. */
  upstream?: VariantStatus
  packages: PackageInfo[]
  /** The runs below are filtered to this variant. */
  selected: boolean
}>()

const emit = defineEmits<{ select: [] }>()

const now = useNow()
const logViewer = useLogViewer()

const pills: Record<HealthState, { tone: PillTone; text: string }> = {
  healthy: { tone: 'success', text: 'Healthy' },
  failing: { tone: 'error', text: 'Failing' },
  building: { tone: 'running', text: 'Building' },
  idle: { tone: 'warning', text: 'No builds' },
}
const pill = computed(() => pills[props.health.state])

const ago = (iso?: string) => formatAgo(iso, now.value)
const packagesKernel = computed(() => {
  const job = props.health.lastSuccess
  return job ? packagesForJob(job, props.packages)[0]?.kernelVersion : undefined
})
</script>

<template>
  <article class="health" :class="[health.state, { selected }]">
    <header>
      <strong>{{ variantLabel(health.variant) }}</strong>
      <StatusPill :tone="pill.tone">{{ pill.text }}</StatusPill>
    </header>

    <div v-if="health.recent.length" class="strip" aria-label="Latest builds, oldest to newest">
      <button
        v-for="r in health.recent"
        :key="r.name"
        type="button"
        class="run"
        :class="r.status"
        :title="`${buildTarget(r)} — ${r.status}, ${formatDuration(r.durationSec)}, ${ago(r.endTime ?? r.startTime)}. Click to open the log.`"
        @click="logViewer.open(r.name)"
      />
      <button v-if="health.active" type="button" class="run Running" :title="`${buildTarget(health.active)} — running. Click to watch the log.`" @click="logViewer.open(health.active.name)" />
    </div>

    <dl>
      <dt>Last success</dt>
      <dd>
        <template v-if="health.lastSuccess">
          {{ ago(health.lastSuccess.endTime) }}
          <span class="muted">· {{ buildTarget(health.lastSuccess) }}</span>
        </template>
        <span v-else class="muted">none on record</span>
      </dd>
      <dt>Success rate</dt>
      <dd>
        <template v-if="health.successRate !== null">{{ health.successRate }}% <span class="muted">of {{ health.runs }}</span></template>
        <span v-else class="muted">-</span>
      </dd>
      <dt>Typical build</dt>
      <dd>{{ health.medianSec ? formatDuration(health.medianSec) : '-' }}</dd>
      <template v-if="upstream">
        <dt>Repository</dt>
        <dd>
          <span v-if="upstream.upToDate" class="ok-text">up to date</span>
          <RouterLink v-else :to="{ name: 'dashboard' }" class="behind" title="Show the upstream sync state">
            behind upstream {{ upstream.upstreamKernel }} &rarr;
          </RouterLink>
        </dd>
      </template>
    </dl>

    <footer>
      <AppButton v-if="health.lastRun" size="small" @click="logViewer.open(health.lastRun.name)">Latest log</AppButton>
      <AppButton v-if="packagesKernel" size="small" :to="{ name: 'dashboard', query: { q: packagesKernel } }">Packages</AppButton>
      <AppButton v-if="health.runs" size="small" :variant="selected ? 'primary' : 'secondary'" @click="emit('select')">
        {{ selected ? 'Showing its runs' : 'Show runs' }}
      </AppButton>
    </footer>
  </article>
</template>

<style scoped>
.health {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px 18px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-top: 3px solid var(--border-color);
  border-radius: 8px;
}

.health.healthy {
  border-top-color: var(--status-success);
}

.health.failing {
  border-top-color: var(--status-error);
}

.health.building {
  border-top-color: var(--accent-blue);
}

.health.selected {
  border-color: var(--accent-blue);
}

header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.strip {
  display: flex;
  gap: 4px;
}

.run {
  width: 14px;
  height: 14px;
  padding: 0;
  border: none;
  border-radius: 3px;
  cursor: pointer;
  background: var(--status-success);
}

.run.Failed {
  background: var(--status-error);
}

.run.Running {
  background: var(--accent-blue);
  animation: pulse 2s infinite;
}

.run:hover,
.run:focus-visible {
  outline: 2px solid var(--text-primary);
  outline-offset: 1px;
}

dl {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: 4px 12px;
  font-size: 0.8125rem;
}

dt {
  color: var(--text-secondary);
}

dd {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.ok-text {
  color: var(--status-success);
}

.behind {
  color: var(--status-warning);
  text-decoration: none;
}

.behind:hover {
  text-decoration: underline;
}

footer {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: auto;
}

@keyframes pulse {
  50% {
    opacity: 0.5;
  }
}
</style>
