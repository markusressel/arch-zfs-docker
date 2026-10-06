<script setup lang="ts">
import { computed, ref } from 'vue'
import { useBuilds } from '../../composables/useBuilds'
import { useDialogs } from '../../composables/useDialogs'
import { useSettings } from '../../composables/useSettings'
import { useUpstream } from '../../composables/useUpstream'
import { describeInterval } from '../../utils/format'
import AppButton from '../ui/AppButton.vue'
import AppCard from '../ui/AppCard.vue'
import KernelStatusItem from './KernelStatusItem.vue'

const { variants, checkNow } = useUpstream()
const { builds, isActive, refresh: refreshBuilds } = useBuilds()
const { settings } = useSettings()
const { alert } = useDialogs()

const checking = ref(false)

const lastChecked = computed(() => {
  const newest = variants.value.reduce((max, v) => (v.checkedAt > max ? v.checkedAt : max), '')
  return newest ? `Last checked ${new Date(newest).toLocaleString()}` : 'Waiting for first upstream check...'
})

const isBuilding = (variant: string) => builds.value.some((b) => isActive(b) && (b.variant || '') === (variant || ''))

async function onCheckNow() {
  checking.value = true
  try {
    await checkNow()
    await refreshBuilds()
  } catch (err) {
    await alert('Check failed', err instanceof Error ? err.message : String(err))
  } finally {
    checking.value = false
  }
}
</script>

<template>
  <AppCard title="Upstream Sync">
    <template #subtitle>{{ lastChecked }}</template>
    <template #actions>
      <AppButton size="small" :disabled="checking" @click="onCheckNow">{{ checking ? 'Checking...' : 'Check Now' }}</AppButton>
    </template>

    <div class="grid">
      <KernelStatusItem v-for="v in variants" :key="v.kernelPkg" :status="v" :building="isBuilding(v.variant)" />
    </div>
    <p class="muted small info">
      Compares the official Arch Linux kernel packages (<code>linux</code>, <code>linux-lts</code>) from <code>archlinux.org</code> with the newest ZFS
      package built here. Polled {{ describeInterval(settings?.autoCheckInterval) }}; a build job is dispatched automatically when a new kernel is released.
    </p>
  </AppCard>
</template>

<style scoped>
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 16px;
}

.info {
  margin-top: 14px;
  line-height: 1.5;
}
</style>
