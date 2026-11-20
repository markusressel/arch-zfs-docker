<script setup lang="ts">
defineProps<{ title?: string }>()
</script>

<template>
  <v-card class="app-card mb-6" color="surface">
    <div v-if="title || $slots.subtitle || $slots.actions" class="d-flex align-start justify-space-between flex-wrap ga-3 pa-5 pb-0">
      <div>
        <h2 v-if="title" class="text-title-medium font-weight-bold">{{ title }}</h2>
        <p v-if="$slots.subtitle" class="text-body-medium text-medium-emphasis">
          <slot name="subtitle" />
        </p>
      </div>
      <div v-if="$slots.actions" class="d-flex align-center flex-wrap ga-2">
        <slot name="actions" />
      </div>
    </div>
    <div class="pa-5"><slot /></div>
  </v-card>
</template>
