<script setup lang="ts">
import { computed } from 'vue'
import type { VariantStatus } from '../../types/api'
import StatusPill from '../ui/StatusPill.vue'

const props = defineProps<{ status: VariantStatus; building: boolean }>()

const pill = computed(() => {
  if (props.status.upToDate) return { tone: 'success', text: 'Up to date' } as const
  if (props.building) return { tone: 'running', text: 'Building' } as const
  return { tone: 'warning', text: 'Build needed' } as const
})
</script>

<template>
  <div class="item" :class="status.upToDate ? 'ok' : 'behind'">
    <div class="head">
      <strong>{{ status.kernelPkg }}</strong>
      <StatusPill :tone="pill.tone">{{ pill.text }}</StatusPill>
    </div>
    <div class="row"><span>Arch upstream</span><code>{{ status.upstreamKernel }}</code></div>
    <div class="row"><span>Our repository</span><code>{{ status.localKernel || 'none built' }}</code></div>
  </div>
</template>

<style scoped>
.item {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-left: 3px solid var(--text-muted);
  border-radius: 6px;
  padding: 14px 16px;
}

.ok {
  border-left-color: var(--status-success);
}

.behind {
  border-left-color: var(--status-warning);
}

.head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  font-size: 0.8125rem;
  padding: 3px 0;
}

.row span {
  color: var(--text-secondary);
}
</style>
