import { computed, ref } from 'vue'
import { fetchPackages } from '../api/repo'
import type { PackageInfo } from '../types/api'

const packages = ref<PackageInfo[]>([])

async function refresh() {
  try {
    packages.value = (await fetchPackages()).packages ?? []
  } catch (err) {
    console.error('Failed to fetch packages:', err)
  }
}

export function usePackages() {
  return { packages, count: computed(() => packages.value.length), refresh }
}
