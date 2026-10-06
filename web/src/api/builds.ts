import type { AlreadyBuilt, BuildRequest, JobSummary } from '../types/api'
import { ApiError, getJson, jsonBody, send } from './http'

export const listBuilds = () => getJson<JobSummary[]>('/api/builds')

export type TriggerResult =
  | { kind: 'started'; job: JobSummary }
  | { kind: 'already-built'; info: AlreadyBuilt }

/** Starts a build; resolves to `already-built` if the target exists and `forceBuild` is false. */
export async function triggerBuild(req: BuildRequest): Promise<TriggerResult> {
  try {
    const res = await send('/api/builds', jsonBody('POST', req))
    return { kind: 'started', job: (await res.json()) as JobSummary }
  } catch (err) {
    if (err instanceof ApiError && err.status === 409) {
      return { kind: 'already-built', info: JSON.parse(err.message) as AlreadyBuilt }
    }
    throw err
  }
}

export async function deleteBuild(name: string): Promise<void> {
  try {
    await send(`/api/builds/${encodeURIComponent(name)}`, { method: 'DELETE' })
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return // already gone
    throw err
  }
}

export async function deleteFinishedBuilds(): Promise<number> {
  const res = await send('/api/builds', { method: 'DELETE' })
  return ((await res.json()) as { deleted: number }).deleted
}

export const logStreamUrl = (jobName: string) => `/api/builds/${encodeURIComponent(jobName)}/logs`
