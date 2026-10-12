<script setup lang="ts">
import { computed } from 'vue'
import { useLogViewer } from '../../composables/useLogViewer'
import type { CompatInfo } from '../../types/api'
import AppButton from '../ui/AppButton.vue'

const props = defineProps<{ compat: CompatInfo; kernel: string }>()
const logViewer = useLogViewer()

const incompatible = computed(() => props.compat.state === 'incompatible')
const visible = computed(() => incompatible.value || props.compat.state === 'predicted_unsupported')
const zfs = computed(() => `OpenZFS ${props.compat.zfsVersion ?? ''}`.trim())
</script>

<template>
  <div v-if="visible" class="notice" :class="{ blocked: incompatible }" role="status">
    <template v-if="incompatible">
      <strong>{{ zfs }} does not build against kernel {{ kernel }}.</strong>
      <p>
        {{ compat.reason }}. Automatic builds for this kernel are paused and resume as soon as a newer OpenZFS release is packaged in the AUR
        or a newer kernel is released.
      </p>
      <AppButton v-if="compat.job" size="small" @click="logViewer.open(compat.job)">Show failed build log</AppButton>
    </template>
    <template v-else>
      <strong>Heads up: {{ zfs }} officially supports Linux up to {{ compat.linuxMax }}.</strong>
      <p>Kernel {{ kernel }} is newer, so the build may fail. It is attempted anyway; a failure pauses retries until a new OpenZFS release.</p>
    </template>

    <div v-if="compat.issues?.length || compat.issuesUrl" class="issues">
      <span class="muted small">Related OpenZFS issues:</span>
      <ul v-if="compat.issues?.length">
        <li v-for="issue in compat.issues" :key="issue.number">
          <a :href="issue.url" target="_blank" rel="noopener">#{{ issue.number }} {{ issue.title }}</a>
          <span class="state" :class="issue.state">{{ issue.state }}</span>
        </li>
      </ul>
      <a v-if="compat.issuesUrl" :href="compat.issuesUrl" target="_blank" rel="noopener" class="small">Search on GitHub</a>
    </div>
  </div>
</template>

<style scoped>
.notice {
  margin-top: 12px;
  padding: 10px 12px;
  border-radius: 6px;
  border: 1px solid rgba(210, 153, 34, 0.4);
  background: rgba(210, 153, 34, 0.08);
  font-size: 0.8125rem;
  line-height: 1.45;
}

.notice.blocked {
  border-color: rgba(248, 81, 73, 0.4);
  background: rgba(248, 81, 73, 0.08);
}

.notice p {
  margin: 4px 0 8px;
  color: var(--text-secondary);
}

.issues {
  margin-top: 8px;
}

.issues ul {
  margin: 4px 0;
  padding-left: 18px;
}

.state {
  margin-left: 6px;
  font-size: 0.6875rem;
  color: var(--text-muted);
}

.state.open {
  color: var(--status-success);
}
</style>
