import { ref, watch } from 'vue'
import { formatDate, formatDateTime, formatTime, normalizePref, type DateFormatPref } from '../utils/datetime'

const STORAGE_KEY = 'dateFormat'

function stored(): string | null {
  try {
    return localStorage.getItem(STORAGE_KEY)
  } catch {
    return null
  }
}

// A display preference of this browser, not a server setting: every visitor may prefer another format.
const pref = ref<DateFormatPref>(normalizePref(stored()))
watch(pref, (p) => {
  try {
    localStorage.setItem(STORAGE_KEY, p)
  } catch {
    // Storage unavailable (e.g. disabled cookies): the choice lasts until the page is reloaded.
  }
})

/** Date formatters following the user's chosen format (the browser's locale by default). */
export function useDateFormat() {
  return {
    pref,
    dateTime: (value: string | number | Date) => formatDateTime(value, pref.value),
    date: (value: string | number | Date) => formatDate(value, pref.value),
    time: (value: string | number | Date, seconds = true) => formatTime(value, pref.value, seconds),
  }
}
