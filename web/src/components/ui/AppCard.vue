<script setup lang="ts">
defineProps<{ title?: string }>()
</script>

<template>
  <section class="card">
    <header v-if="title || $slots.subtitle || $slots.actions" class="head">
      <div>
        <h2 v-if="title">{{ title }}</h2>
        <p v-if="$slots.subtitle" class="muted small"><slot name="subtitle" /></p>
      </div>
      <div v-if="$slots.actions" class="actions"><slot name="actions" /></div>
    </header>
    <slot />
  </section>
</template>

<style scoped>
.card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 18px 20px;
}

.head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 16px;
}

h2 {
  font-size: 1.1rem;
}

.actions {
  display: flex;
  gap: 8px;
}
</style>
