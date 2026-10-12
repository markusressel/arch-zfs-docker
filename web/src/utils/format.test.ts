import { describe, expect, it } from 'vitest'
import { describeInterval, formatAgo, formatBytes, formatDuration } from './format'

describe('formatDuration', () => {
  it('pads to a stable width', () => {
    expect(formatDuration(5)).toBe('00:05')
    expect(formatDuration(65)).toBe('01:05')
    expect(formatDuration(3725)).toBe('1:02:05')
  })

  it('handles invalid input', () => {
    expect(formatDuration(-1)).toBe('-')
    expect(formatDuration(NaN)).toBe('-')
  })
})

describe('formatBytes', () => {
  it('formats with binary units', () => {
    expect(formatBytes(512)).toBe('512 B')
    expect(formatBytes(1536)).toBe('1.50 KiB')
    expect(formatBytes(5 * 1024 * 1024)).toBe('5.00 MiB')
  })
})

describe('describeInterval', () => {
  it('describes intervals', () => {
    expect(describeInterval('6h')).toBe('every 6 hours')
    expect(describeInterval('1h')).toBe('every 1 hour')
    expect(describeInterval('0')).toContain('disabled')
    expect(describeInterval(undefined)).toContain('disabled')
  })
})

describe('formatAgo', () => {
  const now = Date.parse('2026-10-12T12:00:00Z')
  const ago = (sec: number) => new Date(now - sec * 1000).toISOString()

  it('picks a readable unit', () => {
    expect(formatAgo(ago(20), now)).toBe('just now')
    expect(formatAgo(ago(5 * 60), now)).toBe('5 min ago')
    expect(formatAgo(ago(3 * 3600), now)).toBe('3 h ago')
    expect(formatAgo(ago(50 * 3600), now)).toBe('2 d ago')
  })

  it('handles missing or future times', () => {
    expect(formatAgo(undefined, now)).toBe('-')
    expect(formatAgo(ago(-30), now)).toBe('just now')
  })
})
