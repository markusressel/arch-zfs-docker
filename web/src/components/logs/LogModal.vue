<script setup lang="ts">
import { computed, ref, toRef } from 'vue'
import { useLogStream } from '../../composables/useLogStream'
import AppButton from '../ui/AppButton.vue'
import AppModal from '../ui/AppModal.vue'
import LogTerminal from './LogTerminal.vue'

const props = defineProps<{ jobName: string }>()
const emit = defineEmits<{ close: [] }>()

const { lines } = useLogStream(toRef(props, 'jobName'))
const terminal = ref<InstanceType<typeof LogTerminal> | null>(null)
const showJump = computed(() => terminal.value?.following === false)
</script>

<template>
  <AppModal dark max-width="900px" @close="emit('close')">
    <template #header>
      <strong class="title">Logs: {{ jobName }}</strong>
      <div class="buttons">
        <AppButton v-if="showJump" size="small" @click="terminal?.scrollToBottom()">Jump to Bottom &darr;</AppButton>
        <AppButton size="small" @click="emit('close')">Close</AppButton>
      </div>
    </template>
    <LogTerminal ref="terminal" :lines="lines" />
  </AppModal>
</template>

<style scoped>
.title {
  font-family: var(--font-mono);
  font-size: 0.875rem;
}

.buttons {
  display: flex;
  gap: 8px;
}
</style>
