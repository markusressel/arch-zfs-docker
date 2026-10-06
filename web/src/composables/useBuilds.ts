import { computed, ref } from 'vue'
import { deleteBuild, deleteFinishedBuilds, listBuilds } from '../api/builds'
import type { JobSummary } from '../types/api'

const builds = ref<JobSummary[]>([])

const isFinished = (b: JobSummary) => b.status === 'Succeeded' || b.status === 'Failed'
const isActive = (b: JobSummary) => b.status === 'Running' || b.status === 'Pending'

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
    refresh,
    remove,
    clearFinished,
    isFinished,
    isActive,
  }
}
