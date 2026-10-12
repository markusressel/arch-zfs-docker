<script setup lang="ts">
import { computed } from 'vue'
import { iniTokens } from '../../utils/pacman'
import CopyButton from './CopyButton.vue'

const props = defineProps<{
  text: string
  /** "ini" highlights sections, keys and values; "shell" prefixes each line with an unselectable prompt. */
  lang?: 'ini' | 'shell'
  /** Shown in the header bar, e.g. the file the snippet belongs in. */
  caption?: string
}>()

const lines = computed(() => props.text.split('\n'))
</script>

<template>
  <div class="code-block">
    <div class="bar">
      <span class="caption">{{ caption }}</span>
      <CopyButton :text="text" />
    </div>
    <pre><code><template v-for="(line, i) in lines" :key="i"><span v-if="lang === 'shell'" class="prompt" aria-hidden="true">$ </span><template v-if="lang === 'ini'"><span v-for="(t, j) in iniTokens(line)" :key="j" :class="t.kind">{{ t.text }}</span></template><template v-else>{{ line }}</template>{{ i < lines.length - 1 ? '\n' : '' }}</template></code></pre>
  </div>
</template>

<style scoped>
.code-block {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  overflow: hidden;
}

.bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  padding: 6px 8px 6px 12px;
  background: var(--bg-tertiary);
  border-bottom: 1px solid var(--border-color);
}

.caption {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  color: var(--text-secondary);
}

pre {
  margin: 0;
  padding: 12px 14px;
  overflow-x: auto;
  font-family: var(--font-mono);
  font-size: 0.8125rem;
  line-height: 1.6;
  color: var(--text-primary);
}

.prompt,
.punct,
.comment {
  color: var(--text-muted);
}

.prompt {
  user-select: none;
}

.section {
  color: var(--status-success);
}

.key {
  color: var(--accent-blue);
}
</style>
