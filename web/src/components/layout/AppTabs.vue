<script setup lang="ts">
export interface Tab {
  /** Route name */
  name: string
  label: string
  /** Optional counter shown next to the label (hidden when 0). */
  badge?: number
}

defineProps<{ tabs: Tab[] }>()
</script>

<template>
  <nav class="tabs d-flex align-center flex-wrap mb-6">
    <v-tabs color="primary" density="compact">
      <v-tab v-for="tab in tabs" :key="tab.name" :to="{ name: tab.name }" class="text-none">
        {{ tab.label }}
        <v-chip v-if="tab.badge" class="badge ms-2" color="primary" size="x-small">{{ tab.badge }}</v-chip>
      </v-tab>
    </v-tabs>
    <span
      class="live ms-auto text-body-small text-medium-emphasis"
      title="Builds, packages and sync state are fetched in the background (every 2s while a build runs)."
    >
      <span class="dot" />Live: updates automatically, no need to refresh
    </span>
  </nav>
</template>

<style scoped>
.tabs {
  border-bottom: 1px solid var(--border-color);
}

.live {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--status-success);
  animation: pulse 2s infinite;
}

.badge {
  animation: pulse 2s infinite;
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
