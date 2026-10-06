<script setup lang="ts">
import { useSettings } from '../../composables/useSettings'

const { settings } = useSettings()
</script>

<template>
  <header class="app-header">
    <div class="brand">
      <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#58a6ff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
        <polyline points="3.27 6.96 12 12.01 20.73 6.96" />
        <line x1="12" y1="22.08" x2="12" y2="12" />
      </svg>
      <h1>Arch Linux ZFS Repository</h1>
    </div>
    <span class="badge">{{ settings?.repoName || 'zfslocal' }} • x86_64</span>
  </header>
</template>

<style scoped>
.app-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--border-color);
  margin-bottom: 16px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

h1 {
  font-size: 1.5rem;
  font-weight: 600;
  letter-spacing: -0.02em;
}

.badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--status-success);
  border: 1px solid rgba(63, 185, 80, 0.4);
  background: var(--bg-secondary);
}

.badge::before {
  content: '';
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--status-success);
}
</style>
