<script setup lang="ts">
import { computed } from 'vue'
import { useUpstream } from '../../composables/useUpstream'
import type { VariantStatus } from '../../types/api'
import { heldUpdateCommand } from '../../utils/pacman'
import AppCard from '../ui/AppCard.vue'
import CodeBlock from '../ui/CodeBlock.vue'

const { variants } = useUpstream()

// Kernels without a zfs-linux package for the newest upstream version, whatever the reason.
const behind = computed(() => variants.value.filter((v) => !v.upToDate && v.localKernel))

// The error pacman prints for the first such kernel. The dependency names the kernel the installed or
// packaged zfs-linux was built for, which differs per machine.
const example = computed(() => {
  const v = behind.value[0]
  return v ? `:: installing ${v.kernelPkg} (${v.upstreamKernel}) breaks dependency '${v.kernelPkg}=...' required by zfs-${v.kernelPkg}` : ''
})

const command = (v: VariantStatus) => heldUpdateCommand(v.kernelPkg)
</script>

<template>
  <AppCard v-if="behind.length" title="Updating while ZFS is held back">
    <template #subtitle>pacman -Syu fails until the package for the new kernel is built</template>
    <p class="small">
      <code>zfs-linux</code> depends on the exact kernel version it was built for. As long as this repository has no package for the newest Arch kernel, a full
      upgrade stops with an error like this one (or <code>unable to satisfy dependency</code>):
    </p>
    <CodeBlock :text="example" caption="pacman" />
    <div v-for="v in behind" :key="v.kernelPkg" class="variant">
      <p class="small">
        If your machine runs <code>{{ v.kernelPkg }}</code> (newest upstream <code>{{ v.upstreamKernel }}</code>, built here <code>{{ v.localKernel }}</code>),
        update everything except the kernel and ZFS:
      </p>
      <CodeBlock :text="command(v)" lang="shell" caption="Update without the kernel" />
    </div>
    <p class="small">
      Once the package is built (this card disappears), run a normal
      <code>sudo pacman -Syu</code> to update the kernel and ZFS together. Do not add these packages to <code>IgnorePkg</code> in <code>/etc/pacman.conf</code>:
      then the kernel stops updating for good.
    </p>
  </AppCard>
</template>

<style scoped>
.variant {
  margin-top: 12px;
}

p {
  margin: 8px 0;
  line-height: 1.5;
}
</style>
