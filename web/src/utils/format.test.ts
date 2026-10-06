import { describe, expect, it } from 'vitest'
import { describeInterval, formatBytes, formatDuration } from './format'

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
