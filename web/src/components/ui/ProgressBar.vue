<script setup lang="ts">
defineProps<{ percent: number }>()
</script>

<template>
  <div class="progress"><div :style="{ width: `${percent}%` }" /></div>
</template>

<style scoped>
.progress {
  height: 4px;
  width: 100%;
  min-width: 90px;
  background: var(--bg-tertiary);
  border-radius: 2px;
  margin-top: 4px;
  overflow: hidden;
}

.progress > div {
  height: 100%;
  background: var(--accent-blue);
  transition: width 1s linear;
}
</style>
