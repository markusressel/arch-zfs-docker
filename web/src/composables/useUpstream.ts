import { computed, ref } from 'vue'
import { fetchUpstream, triggerUpstreamCheck } from '../api/repo'
import type { UpstreamStatus } from '../types/api'

const status = ref<UpstreamStatus>({})

async function refresh() {
  try {
    status.value = (await fetchUpstream()) ?? {}
  } catch (err) {
    console.error('Failed to fetch upstream status:', err)
  }
}

/** Triggers a check on the server and reloads the result shortly after. */
async function checkNow() {
  await triggerUpstreamCheck()
  await new Promise((resolve) => setTimeout(resolve, 2000))
  await refresh()
}

export function useUpstream() {
  return {
    status,
    variants: computed(() => Object.values(status.value).sort((a, b) => a.kernelPkg.localeCompare(b.kernelPkg))),
    refresh,
    checkNow,
  }
}
