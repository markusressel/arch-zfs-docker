<script setup lang="ts">
import BuildForm from '../components/dashboard/BuildForm.vue'
import BuildTable from '../components/dashboard/BuildTable.vue'
</script>

<template>
  <div class="main-grid">
    <BuildForm />
    <BuildTable />
  </div>
</template>

<style scoped>
.main-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 24px;
  align-items: start;
}

@media (min-width: 900px) {
  .main-grid {
    grid-template-columns: 360px 1fr;
  }
}
</style>
