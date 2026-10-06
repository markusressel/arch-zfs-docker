<script setup lang="ts">
export type PillTone = 'success' | 'running' | 'error' | 'warning'

defineProps<{ tone: PillTone }>()
</script>

<template>
  <span class="pill" :class="tone"><slot /></span>
</template>

<style scoped>
.pill {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
}

.success {
  background: rgba(63, 185, 80, 0.15);
  color: var(--status-success);
}

.running {
  background: rgba(88, 166, 255, 0.15);
  color: var(--accent-blue);
  animation: pulse 2s infinite;
}

.error {
  background: rgba(248, 81, 73, 0.15);
  color: var(--status-error);
}

.warning {
  background: rgba(210, 153, 34, 0.15);
  color: var(--status-warning);
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.6;
  }
}
</style>
