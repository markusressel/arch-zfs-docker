<script setup lang="ts">
import ClientConfigBox from '../components/dashboard/ClientConfigBox.vue'
import KernelStatusCard from '../components/dashboard/KernelStatusCard.vue'
import PackageCatalog from '../components/dashboard/PackageCatalog.vue'
</script>

<template>
  <div class="dashboard">
    <KernelStatusCard />
    <ClientConfigBox />
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
