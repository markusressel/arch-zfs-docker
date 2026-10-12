<script setup lang="ts">
import KernelStatusCard from '../components/dashboard/KernelStatusCard.vue'
import PackageCatalog from '../components/dashboard/PackageCatalog.vue'
</script>

<template>
  <div class="dashboard">
    <KernelStatusCard />
    <PackageCatalog />
  </div>
</template>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 24px;
}
</style>
