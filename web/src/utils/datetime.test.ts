import { expect, it } from 'vitest'
import { AUTO, formatDate, formatDateTime, formatTime, ISO, localeChoices, normalizePref } from './datetime'

// Local time, so the expectations do not depend on the time zone of the test machine.
const at = new Date(2026, 9, 2, 15, 4, 5)

it('formats ISO 8601 in local time', () => {
  expect(formatDateTime(at, ISO)).toBe('2026-10-02 15:04:05')
  expect(formatDate(at, ISO)).toBe('2026-10-02')
  expect(formatTime(at, ISO)).toBe('15:04:05')
  expect(formatTime(at, ISO, false)).toBe('15:04')
})

it('formats with an explicit locale', () => {
  expect(formatDate(at, 'de-DE')).toBe('2.10.2026')
  expect(formatDate(at, 'en-US')).toBe('10/2/2026')
  expect(formatTime(at, 'de-DE', false)).toBe('15:04')
})

it('formats with the browser locale for auto', () => {
  expect(formatDateTime(at, AUTO)).toBe(at.toLocaleString())
})

it('shows a dash for invalid dates', () => {
  expect(formatDateTime('not a date', ISO)).toBe('-')
  expect(formatDate('', AUTO)).toBe('-')
})

it('falls back to auto for unknown preferences', () => {
  expect(normalizePref(null)).toBe(AUTO)
  expect(normalizePref('iso')).toBe(ISO)
  expect(normalizePref('de-DE')).toBe('de-DE')
  expect(normalizePref('not a locale!')).toBe(AUTO)
})

it('lists the browser languages first, without duplicates', () => {
  const choices = localeChoices(['de-DE', 'en-US'])
  expect(choices.slice(0, 2)).toEqual(['de-DE', 'en-US'])
  expect(choices.filter((c) => c === 'en-US')).toHaveLength(1)
  expect(choices).toContain('en-GB')
})
