<script setup lang="ts">
export interface Tab {
  /** Route name */
  name: string
  label: string
}

defineProps<{ tabs: Tab[] }>()
</script>

<template>
  <nav class="tabs">
    <RouterLink v-for="tab in tabs" :key="tab.name" :to="{ name: tab.name }" class="tab" active-class="active" exact-active-class="active">
      {{ tab.label }}
    </RouterLink>
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
