<script setup lang="ts">
export type PillTone = 'success' | 'running' | 'error' | 'warning'

defineProps<{ tone: PillTone }>()

const COLORS: Record<PillTone, string> = {
  success: 'success',
  running: 'info',
  error: 'error',
  warning: 'warning',
}
</script>

<template>
  <v-chip class="pill font-weight-medium" :color="COLORS[tone]" label><slot /></v-chip>
</template>
