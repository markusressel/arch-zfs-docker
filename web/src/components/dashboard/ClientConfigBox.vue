<script setup lang="ts">
import { computed } from 'vue'
import { useSettings } from '../../composables/useSettings'
import { pacmanSnippet } from '../../utils/pacman'
import CopyButton from '../ui/CopyButton.vue'

const { settings } = useSettings()
const snippet = computed(() => pacmanSnippet(settings.value?.repoName, window.location.origin))
</script>

<template>
  <div class="box">
    <div><strong>Client Configuration:</strong> Add this repository to your <code>/etc/pacman.conf</code>:</div>
    <div class="snippet-row">
      <pre class="snippet">{{ snippet }}</pre>
      <CopyButton :text="snippet" />
    </div>
  </div>
</template>

<style scoped>
.box {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 16px 20px;
}

.snippet-row {
  display: flex;
  gap: 8px;
  align-items: flex-start;
}

.snippet {
  margin: 0;
  padding: 6px 12px;
  font-family: var(--font-mono);
  font-size: 0.8125rem;
  line-height: 1.5;
  color: var(--accent-blue);
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
}
</style>
