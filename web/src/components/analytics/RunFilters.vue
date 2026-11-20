<script setup lang="ts">
import { computed } from 'vue'
import type { RunFilter, RunStatus } from '../../utils/analytics'
import { KNOWN_VARIANTS, variantLabel } from '../../utils/jobs'
import AppButton from '../ui/AppButton.vue'
import SegmentedControl, { type Segment } from '../ui/SegmentedControl.vue'

const model = defineModel<RunFilter>({ required: true })

const variants: Segment<string | undefined>[] = [
  { value: undefined, label: 'All kernels' },
  ...KNOWN_VARIANTS.map((v) => ({ value: v, label: variantLabel(v) })),
]
const statuses: Segment<RunStatus | undefined>[] = [
  { value: undefined, label: 'All results' },
  { value: 'Succeeded', label: 'Succeeded' },
  { value: 'Failed', label: 'Failed' },
]

const variant = computed({
  get: () => model.value.variant,
  set: (variant) => (model.value = { ...model.value, variant }),
})
const status = computed({
  get: () => model.value.status,
  set: (status) => (model.value = { ...model.value, status }),
})
const active = computed(() => Object.values(model.value).some((v) => v !== undefined))
</script>

<template>
  <div class="filters">
    <SegmentedControl v-model="variant" :options="variants" label="Kernel" />
    <SegmentedControl v-model="status" :options="statuses" label="Result" />
    <v-chip v-if="model.bucket" color="primary" closable close-label="Remove the duration filter" @click:close="model = { ...model, bucket: undefined }">
      Duration {{ model.bucket }}
    </v-chip>
    <AppButton v-if="active" size="small" @click="model = {}">Reset filters</AppButton>
  </div>
</template>

<style scoped>
.filters {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}
</style>
