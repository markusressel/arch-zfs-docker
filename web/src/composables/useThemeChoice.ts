import { ref, watch } from 'vue'
import type { ThemeChoice } from '../plugins/vuetify'

const STORAGE_KEY = 'theme'
const CHOICES: ThemeChoice[] = ['system', 'light', 'dark']

function stored(): ThemeChoice {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    return CHOICES.includes(v as ThemeChoice) ? (v as ThemeChoice) : 'system'
  } catch {
    return 'system'
  }
}

// A display preference of this browser, like the date format. App.vue applies it to Vuetify.
const choice = ref<ThemeChoice>(stored())
watch(choice, (c) => {
  try {
    localStorage.setItem(STORAGE_KEY, c)
  } catch {
    // Storage unavailable: the choice lasts until the page is reloaded.
  }
})

/** The color theme of this browser: the system's, light or dark. */
export function useThemeChoice() {
  return { choice, choices: CHOICES }
}
