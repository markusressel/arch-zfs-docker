import { ref } from 'vue'
import { fetchSettings, saveSettings } from '../api/repo'
import type { Settings, SettingsUpdate } from '../types/api'

const settings = ref<Settings | null>(null)

async function load() {
  try {
    settings.value = await fetchSettings()
  } catch (err) {
    console.error('Failed to fetch settings:', err)
  }
}

async function save(update: SettingsUpdate) {
  settings.value = await saveSettings(update)
}

export function useSettings() {
  return { settings, load, save }
}
