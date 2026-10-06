import { ref, watch, type Ref } from 'vue'
import { fetchKernelVersions } from '../api/repo'

/** Loads the installable kernel versions for the selected variant (autocomplete suggestions). */
export function useKernelVersions(variant: Ref<string>) {
  const versions = ref<string[]>([])

  async function load() {
    try {
      versions.value = await fetchKernelVersions(variant.value)
    } catch (err) {
      // The input still accepts free text without suggestions.
      versions.value = []
      console.warn('Failed to fetch kernel versions:', err)
    }
  }

  watch(variant, load, { immediate: true })
  return { versions }
}
