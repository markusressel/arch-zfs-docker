<script setup lang="ts">
withDefaults(defineProps<{ title?: string; maxWidth?: string; dark?: boolean }>(), { maxWidth: '640px' })
const emit = defineEmits<{ close: [] }>()
</script>

<template>
  <!-- Mounted while open; closing (Escape, a click outside) asks the parent to unmount it. -->
  <v-dialog :model-value="true" :max-width="maxWidth" scrollable @update:model-value="(open: boolean) => !open && emit('close')">
    <v-card :theme="dark ? 'dark' : undefined" role="dialog" aria-modal="true">
      <v-card-title class="d-flex align-center ga-2 text-title-medium font-weight-bold">
        <slot name="header">{{ title }}</slot>
      </v-card-title>
      <v-divider />
      <v-card-text class="pa-4"><slot /></v-card-text>
      <template v-if="$slots.footer">
        <v-divider />
        <v-card-actions class="justify-end ga-2 pa-3"><slot name="footer" /></v-card-actions>
      </template>
    </v-card>
  </v-dialog>
</template>
