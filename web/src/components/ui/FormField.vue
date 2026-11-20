<script setup lang="ts">
// A Vuetify field with a hint below it. The hint slot allows markup (code spans), which the hint prop of the
// Vuetify fields does not.
defineProps<{ hint?: string }>()
</script>

<template>
  <div>
    <slot />
    <p v-if="hint || $slots.hint" class="text-body-small text-medium-emphasis mt-1 px-1">
      <slot name="hint">{{ hint }}</slot>
    </p>
  </div>
</template>
