<script setup lang="ts">
import { useDialogs } from '../../composables/useDialogs'
import AppButton from '../ui/AppButton.vue'
import AppModal from '../ui/AppModal.vue'

const { state, resolve } = useDialogs()
</script>

<template>
  <AppModal v-if="state.open" :title="state.title" @close="resolve(false)">
    <div class="body">
      <p>{{ state.message }}</p>
      <code v-if="state.detail">{{ state.detail }}</code>
    </div>
    <template #footer>
      <AppButton v-if="state.cancellable" @click="resolve(false)">Cancel</AppButton>
      <AppButton :variant="state.danger ? 'danger' : 'secondary'" @click="resolve(true)">
        {{ state.confirmLabel || (state.cancellable ? 'Confirm' : 'Close') }}
      </AppButton>
    </template>
  </AppModal>
</template>

<style scoped>
.body {
  padding: 16px;
  font-size: 0.875rem;
  line-height: 1.5;
  overflow-y: auto;
}

code {
  display: block;
  margin-top: 12px;
  padding: 6px 10px;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 0.75rem;
  word-break: break-all;
}
</style>
