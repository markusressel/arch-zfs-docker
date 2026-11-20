<script setup lang="ts" generic="T">
import { computed } from 'vue'

export interface Segment<V> {
  value: V
  label: string
}

const props = defineProps<{ options: Segment<T>[]; label?: string }>()
const model = defineModel<T>({ required: true })

// v-btn-toggle works on the index of the option: option values may be undefined ("All kernels"), which Vuetify
// cannot tell apart from "nothing selected". Clicking the selected option again keeps it selected.
const selected = computed({
  get: () => props.options.findIndex((o) => o.value === model.value),
  set: (i: number | undefined) => {
    if (i !== undefined && i >= 0) model.value = props.options[i]!.value
  },
})
</script>

<template>
  <div class="d-inline-flex align-center ga-2" role="group" :aria-label="label">
    <span v-if="label" class="text-body-small text-medium-emphasis">{{ label }}</span>
    <v-btn-toggle v-model="selected" color="primary">
      <v-btn v-for="(o, i) in options" :key="o.label" :value="i" :aria-pressed="o.value === model" size="small" class="text-none">{{ o.label }}</v-btn>
    </v-btn-toggle>
  </div>
</template>
