const pad2 = (n: number) => String(n).padStart(2, '0')

/** Fixed-width MM:SS (or H:MM:SS) so a ticking column does not jitter. */
export function formatDuration(totalSeconds: number): string {
  if (!Number.isFinite(totalSeconds) || totalSeconds < 0) return '-'
  const s = Math.floor(totalSeconds)
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  return h > 0 ? `${h}:${pad2(m)}:${pad2(sec)}` : `${pad2(m)}:${pad2(sec)}`
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  const units = ['KiB', 'MiB', 'GiB', 'TiB']
  let value = bytes
  let i = -1
  do {
    value /= 1024
    i++
  } while (value >= 1024 && i < units.length - 1)
  return `${value.toFixed(2)} ${units[i]}`
}

/** Human readable description of an auto-check interval such as "6h". */
export function describeInterval(interval: string | undefined): string {
  if (!interval || interval === '0' || interval === '0s') return 'manual trigger only (auto-checking disabled)'
  return `every ${interval.replace(/^1h$/, '1 hour').replace(/^(\d+)h$/, '$1 hours')}`
}
