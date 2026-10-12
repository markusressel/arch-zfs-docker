<script setup lang="ts">
import { computed } from 'vue'
import { useDateFormat } from '../../composables/useDateFormat'
import { AUTO, formatDateTime, ISO, localeChoices } from '../../utils/datetime'
import AppCard from '../ui/AppCard.vue'
import FormField from '../ui/FormField.vue'

const { pref } = useDateFormat()

// Each choice shows what a date looks like with it, which says more than a locale tag.
const sample = new Date()
const browserLocale = new Intl.DateTimeFormat().resolvedOptions().locale
const options = computed(() => [
  { value: AUTO, label: `Browser default (${browserLocale}): ${formatDateTime(sample, AUTO)}` },
  { value: ISO, label: `ISO 8601: ${formatDateTime(sample, ISO)}` },
  ...localeChoices(navigator.languages ?? []).map((l) => ({ value: l, label: `${l}: ${formatDateTime(sample, l)}` })),
])
</script>

<template>
  <AppCard title="Display">
    <template #subtitle>Stored in this browser only.</template>
    <FormField label="Date & time format" for="setting-date-format">
      <select id="setting-date-format" v-model="pref">
        <option v-for="o in options" :key="o.value" :value="o.value">{{ o.label }}</option>
      </select>
      <template #hint>The browser default follows the browser's language, which may differ from your region's format.</template>
    </FormField>
  </AppCard>
</template>
