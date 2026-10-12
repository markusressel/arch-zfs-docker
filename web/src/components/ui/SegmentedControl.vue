<script setup lang="ts" generic="T">
export interface Segment<V> {
  value: V
  label: string
}

defineProps<{ options: Segment<T>[]; label?: string }>()
const model = defineModel<T>({ required: true })
</script>

<template>
  <div class="segmented" role="group" :aria-label="label">
    <span v-if="label" class="label">{{ label }}</span>
    <button
      v-for="o in options"
      :key="o.label"
      type="button"
      :class="{ active: o.value === model }"
      :aria-pressed="o.value === model"
      @click="model = o.value"
    >
      {{ o.label }}
    </button>
  </div>
</template>

<style scoped>
.segmented {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
}

.label {
  padding: 0 8px;
  font-size: 0.75rem;
  color: var(--text-muted);
}

button {
  padding: 4px 10px;
  border: none;
  border-radius: 4px;
  background: none;
  color: var(--text-secondary);
  font-size: 0.8125rem;
  cursor: pointer;
}

button:hover {
  color: var(--text-primary);
}

button.active {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}
</style>
