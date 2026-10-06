/** Number of log lines kept in the terminal scrollback. */
export const MAX_LOG_LINES = 2000

/** Appends `chunk` to `lines`, keeping only the newest `max` lines. Returns a new array. */
export function appendBounded(lines: string[], chunk: string[], max: number = MAX_LOG_LINES): string[] {
  if (chunk.length >= max) return chunk.slice(chunk.length - max)
  const merged = lines.concat(chunk)
  return merged.length > max ? merged.slice(merged.length - max) : merged
}
