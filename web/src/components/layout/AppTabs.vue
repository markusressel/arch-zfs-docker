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
  <nav class="tabs">
    <RouterLink v-for="tab in tabs" :key="tab.name" :to="{ name: tab.name }" class="tab" active-class="active" exact-active-class="active">
      {{ tab.label }}
      <span v-if="tab.badge" class="badge">{{ tab.badge }}</span>
    </RouterLink>
    <span class="live" title="Builds, packages and sync state are fetched in the background (every 2s while a build runs).">
      <span class="dot" />Live: updates automatically, no need to refresh
    </span>
  </nav>
</template>

<style scoped>
.tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 24px;
  border-bottom: 1px solid var(--border-color);
  padding-bottom: 12px;
}

.tab {
  padding: 8px 16px;
  color: var(--text-secondary);
  border: 1px solid transparent;
  border-radius: 6px;
  font-size: 0.875rem;
  font-weight: 500;
  text-decoration: none;
  transition: all 0.15s ease-in-out;
}

.live {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-left: auto;
  align-self: center;
  color: var(--text-muted);
  font-size: 0.75rem;
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--status-success);
  animation: pulse 2s infinite;
}

.badge {
  display: inline-block;
  min-width: 18px;
  margin-left: 6px;
  padding: 0 6px;
  border-radius: 9999px;
  background: rgba(88, 166, 255, 0.15);
  color: var(--accent-blue);
  font-size: 0.75rem;
  font-weight: 600;
  text-align: center;
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

.tab:hover {
  color: var(--text-primary);
  background: var(--bg-tertiary);
}

.tab.active {
  color: var(--text-primary);
  background: var(--bg-secondary);
  border-color: var(--border-color);
  font-weight: 600;
}
</style>
