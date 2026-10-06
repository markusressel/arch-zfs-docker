<script setup lang="ts">
import BuildForm from '../components/dashboard/BuildForm.vue'
import BuildTable from '../components/dashboard/BuildTable.vue'
import ClientConfigBox from '../components/dashboard/ClientConfigBox.vue'
import KernelStatusCard from '../components/dashboard/KernelStatusCard.vue'
import PackageCatalog from '../components/dashboard/PackageCatalog.vue'
</script>

<template>
  <div class="dashboard">
    <KernelStatusCard />
    <ClientConfigBox />
    <div class="main-grid">
      <BuildForm />
      <BuildTable />
    </div>
    <PackageCatalog />
  </div>
</template>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

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
