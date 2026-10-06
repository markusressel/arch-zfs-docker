<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useStickToBottom } from '../../composables/useStickToBottom'

const props = defineProps<{ lines: string[] }>()

const el = ref<HTMLElement | null>(null)
const { following, onScroll, scrollToBottom, contentChanged } = useStickToBottom(el)

// A single text node is far cheaper to patch than thousands of line elements.
const text = computed(() => props.lines.join('\n'))

// Runs after the DOM was updated: stay pinned to the bottom only while the user is following.
watch(text, contentChanged, { flush: 'post' })

defineExpose({ following, scrollToBottom })
</script>

<template>
  <pre ref="el" class="terminal" @scroll.passive="onScroll">{{ text }}</pre>
</template>

<style scoped>
.terminal {
  margin: 0;
  padding: 16px;
  flex: 1;
  min-height: 300px;
  overflow-y: auto;
  font-family: var(--font-mono);
  font-size: 0.8125rem;
  line-height: 1.5;
  color: #e6edf3;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
