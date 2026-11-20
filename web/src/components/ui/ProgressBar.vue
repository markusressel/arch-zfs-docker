<script setup lang="ts">
defineProps<{ percent: number }>()
</script>

<template>
  <v-progress-linear :model-value="percent" height="6" class="my-2" />
</template>
