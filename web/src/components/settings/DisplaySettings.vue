<script setup lang="ts">
import { computed } from 'vue'
import { useDateFormat } from '../../composables/useDateFormat'
import { useThemeChoice } from '../../composables/useThemeChoice'
import { AUTO, formatDateTime, ISO, localeChoices } from '../../utils/datetime'
import AppCard from '../ui/AppCard.vue'
import SegmentedControl from '../ui/SegmentedControl.vue'

const { pref } = useDateFormat()
const { choice: theme } = useThemeChoice()
const themes = [
  { value: 'system' as const, label: 'System' },
  { value: 'light' as const, label: 'Light' },
  { value: 'dark' as const, label: 'Dark' },
]

// Each choice shows what a date looks like with it, which says more than a locale tag.
const sample = new Date()
const browserLocale = new Intl.DateTimeFormat().resolvedOptions().locale
const options = computed(() => [
  {
    value: AUTO,
    label: `Browser default (${browserLocale}): ${formatDateTime(sample, AUTO)}`,
  },
  { value: ISO, label: `ISO 8601: ${formatDateTime(sample, ISO)}` },
  ...localeChoices(navigator.languages ?? []).map((l) => ({
    value: l,
    label: `${l}: ${formatDateTime(sample, l)}`,
  })),
])
</script>

<template>
  <AppCard title="Display">
    <template #subtitle>Stored in this browser only.</template>
    <SegmentedControl v-model="theme" :options="themes" label="Theme" class="mb-5" />
    <v-select
      id="setting-date-format"
      v-model="pref"
      :items="options"
      item-title="label"
      item-value="value"
      label="Date & time format"
      hint="The browser default follows the browser's language, which may differ from your region's format."
      persistent-hint
    />
  </AppCard>
</template>
