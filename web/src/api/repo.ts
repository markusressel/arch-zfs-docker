import type { RepoSummary, Settings, SettingsUpdate, UpstreamStatus } from '../types/api'
import { getJson, jsonBody, send } from './http'

export const fetchPackages = () => getJson<RepoSummary>('/api/packages')

export const fetchUpstream = () => getJson<UpstreamStatus>('/api/upstream')

export const triggerUpstreamCheck = () => send('/api/upstream/check', { method: 'POST' })

export async function fetchKernelVersions(variant: string): Promise<string[]> {
  const data = await getJson<{ versions: string[] }>(`/api/kernels?variant=${encodeURIComponent(variant)}`)
  return data.versions ?? []
}

export async function fetchZfsVersions(): Promise<string[]> {
  return (await getJson<{ versions: string[] }>('/api/zfs-versions')).versions ?? []
}

export const fetchSettings = () => getJson<Settings>('/api/settings')

export async function saveSettings(update: SettingsUpdate): Promise<Settings> {
  return (await send('/api/settings', jsonBody('POST', update))).json() as Promise<Settings>
}
