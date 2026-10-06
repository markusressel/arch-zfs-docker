import { onMounted, ref } from 'vue'
import { fetchZfsVersions } from '../api/repo'

/** Loads released OpenZFS versions (autocomplete suggestions). */
export function useZfsVersions() {
  const versions = ref<string[]>([])

  onMounted(async () => {
    try {
      versions.value = await fetchZfsVersions()
    } catch (err) {
      // The input still accepts free text without suggestions.
      console.warn('Failed to fetch ZFS versions:', err)
    }
  })

  return { versions }
}
