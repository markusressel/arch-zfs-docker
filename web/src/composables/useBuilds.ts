import { computed, ref } from 'vue'
import { deleteBuild, deleteFinishedBuilds, listBuilds } from '../api/builds'
import type { JobSummary } from '../types/api'
import { countActiveBuilds, isFinished } from '../utils/jobs'

const builds = ref<JobSummary[]>([])
/** False until the first successful fetch, so an empty list is not mistaken for "no builds". */
const loaded = ref(false)

async function refresh() {
  try {
    builds.value = (await listBuilds()) ?? []
    loaded.value = true
  } catch (err) {
    console.error('Failed to fetch builds:', err)
  }
}

async function remove(name: string) {
  await deleteBuild(name)
  await refresh()
}

async function clearFinished() {
  await deleteFinishedBuilds()
  await refresh()
}

/** Shared build job state: every component sees the same list. */
export function useBuilds() {
  return {
    builds,
    loaded,
    finishedCount: computed(() => builds.value.filter(isFinished).length),
    /** Running or pending builds; maintenance jobs do not count. */
    activeCount: computed(() => countActiveBuilds(builds.value)),
    refresh,
    remove,
    clearFinished,
  }
}
