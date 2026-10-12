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
  selected?: boolean
}

const props = withDefaults(
  defineProps<{
    bars: Bar[]
    maxBarWidth?: string
    fill?: number
    /** Bars are buttons that emit `select`. */
    clickable?: boolean
    /** Dashed horizontal line, e.g. the median. */
    reference?: { value: number; label: string }
  }>(),
  { maxBarWidth: '48px', fill: 100 },
)

const emit = defineEmits<{ select: [key: string] }>()

const max = computed(() => Math.max(1, props.reference?.value ?? 0, ...props.bars.map((b) => b.value)))
const height = (value: number) => `${(value / max.value) * props.fill}%`
const anySelected = computed(() => props.bars.some((b) => b.selected))
</script>

<template>
  <div class="chart" role="img">
    <div class="plot">
      <div v-if="reference && reference.value > 0" class="reference" :style="{ bottom: height(reference.value) }">
        <span>{{ reference.label }}</span>
      </div>
      <component
        :is="clickable ? 'button' : 'div'"
        v-for="bar in bars"
        :key="bar.key"
        :type="clickable ? 'button' : undefined"
        class="col"
        :class="{ clickable, selected: bar.selected, dimmed: anySelected && !bar.selected }"
        :style="{ maxWidth: maxBarWidth }"
        :title="bar.title"
        :aria-label="clickable ? bar.title : undefined"
        @click="clickable && emit('select', bar.key)"
      >
        <span v-if="bar.showValue" class="value">{{ bar.value }}</span>
        <div class="bar" :class="bar.tone" :style="{ height: height(bar.value) }" />
        <span v-if="bar.label" class="label">{{ bar.label }}</span>
      </component>
    </div>
  </div>
</template>

<style scoped>
.chart {
  margin-top: 16px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--border-color);
}

.plot {
  position: relative;
  display: flex;
  align-items: flex-end;
  gap: 4px;
  height: 180px;
}

.reference {
  position: absolute;
  left: 0;
  right: 0;
  border-top: 1px dashed var(--text-muted);
  pointer-events: none;
  z-index: 1;
}

.reference span {
  position: absolute;
  right: 0;
  top: -18px;
  padding: 0 4px;
  font-size: 0.6875rem;
  color: var(--text-secondary);
  background: var(--bg-secondary);
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
  padding: 0;
  border: none;
  background: none;
  font: inherit;
  color: inherit;
}

.col.clickable {
  cursor: pointer;
}

.col.clickable:hover .bar,
.col.clickable:focus-visible .bar {
  filter: brightness(1.3);
}

.col.dimmed {
  opacity: 0.35;
}

.col.selected .bar {
  outline: 2px solid var(--text-primary);
  outline-offset: 1px;
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
