<script setup lang="ts">
defineProps<{ label: string; for?: string; hint?: string }>()
</script>

<template>
  <div class="field">
    <label :for="$props.for">{{ label }}</label>
    <slot />
    <small v-if="hint || $slots.hint"><slot name="hint">{{ hint }}</slot></small>
  </div>
</template>

<style scoped>
.field {
  margin-bottom: 14px;
}

label {
  display: block;
  font-size: 0.8125rem;
  color: var(--text-secondary);
  margin-bottom: 6px;
}

small {
  display: block;
  margin-top: 4px;
  color: var(--text-muted);
  font-size: 0.75rem;
}
</style>
