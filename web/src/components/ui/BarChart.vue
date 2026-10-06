<script setup lang="ts">
import { computed } from 'vue'

export interface Bar {
  key: string
  value: number
  /** Tooltip text. */
  title?: string
  /** Label below the bar. */
  label?: string
  /** Show the numeric value above the bar. */
  showValue?: boolean
  tone?: 'primary' | 'danger'
}

const props = withDefaults(defineProps<{ bars: Bar[]; maxBarWidth?: string; fill?: number }>(), {
  maxBarWidth: '48px',
  fill: 100,
})

const max = computed(() => Math.max(1, ...props.bars.map((b) => b.value)))
</script>

<template>
  <div class="chart" role="img">
    <div v-for="bar in bars" :key="bar.key" class="col" :style="{ maxWidth: maxBarWidth }" :title="bar.title">
      <span v-if="bar.showValue" class="value">{{ bar.value }}</span>
      <div class="bar" :class="bar.tone" :style="{ height: `${(bar.value / max) * fill}%` }" />
      <span v-if="bar.label" class="label">{{ bar.label }}</span>
    </div>
  </div>
</template>

<style scoped>
.chart {
  display: flex;
  align-items: flex-end;
  gap: 4px;
  height: 180px;
  margin-top: 16px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--border-color);
}

.col {
  flex: 1;
  min-width: 6px;
  height: 100%;
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
  align-items: center;
  position: relative;
}

.bar {
  width: 100%;
  min-height: 2px;
  background: var(--accent-blue);
  border-radius: 3px 3px 0 0;
}

.bar.danger {
  background: var(--status-error);
}

.value {
  margin-bottom: 2px;
  font-size: 0.6875rem;
  color: var(--text-secondary);
  font-variant-numeric: tabular-nums;
}

.label {
  position: absolute;
  bottom: -20px;
  font-size: 0.625rem;
  color: var(--text-muted);
  white-space: nowrap;
}
</style>
