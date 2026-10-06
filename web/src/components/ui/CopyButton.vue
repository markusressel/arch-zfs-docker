<script setup lang="ts">
import { useClipboard } from '../../composables/useClipboard'
import AppButton from './AppButton.vue'

const props = defineProps<{ text: string }>()
const { copied, copy } = useClipboard()
</script>

<template>
  <AppButton size="small" @click="copy(props.text)">{{ copied ? 'Copied' : 'Copy' }}</AppButton>
</template>
