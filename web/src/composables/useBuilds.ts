import { computed, ref } from 'vue'
import { deleteBuild, deleteFinishedBuilds, listBuilds } from '../api/builds'
import type { JobSummary } from '../types/api'
import { countActiveBuilds, isFinished } from '../utils/jobs'

const builds = ref<JobSummary[]>([])

async function refresh() {
  try {
    builds.value = (await listBuilds()) ?? []
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
    finishedCount: computed(() => builds.value.filter(isFinished).length),
    /** Running or pending builds; maintenance jobs do not count. */
    activeCount: computed(() => countActiveBuilds(builds.value)),
    refresh,
    remove,
    clearFinished,
  }
}
