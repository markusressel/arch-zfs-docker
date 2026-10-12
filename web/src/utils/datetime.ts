/**
 * How dates and times are shown: "auto" follows the browser's locale, "iso" is ISO 8601 (2026-10-12 15:30:00),
 * anything else is a BCP 47 locale tag such as "de-DE".
 */
export type DateFormatPref = string

export const AUTO = 'auto'
export const ISO = 'iso'

/** Locales offered in addition to the browser's own languages. */
const COMMON_LOCALES = ['en-US', 'en-GB', 'de-DE', 'fr-FR', 'es-ES', 'it-IT', 'nl-NL', 'pl-PL', 'pt-BR', 'ja-JP', 'zh-CN']

const pad2 = (n: number) => String(n).padStart(2, '0')

/** Falls back to "auto" for anything the browser cannot format, e.g. a stale or hand-edited stored value. */
export function normalizePref(pref: string | null | undefined): DateFormatPref {
  if (!pref || pref === AUTO || pref === ISO) return pref || AUTO
  try {
    return Intl.DateTimeFormat.supportedLocalesOf([pref]).length > 0 ? pref : AUTO
  } catch {
    return AUTO
  }
}

const locale = (pref: DateFormatPref) => (pref === AUTO ? undefined : pref)

const parse = (value: string | number | Date) => (value instanceof Date ? value : new Date(value))

const isoDate = (d: Date) => `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`
const isoTime = (d: Date, seconds: boolean) => `${pad2(d.getHours())}:${pad2(d.getMinutes())}${seconds ? `:${pad2(d.getSeconds())}` : ''}`

/** Date and time, e.g. "10/12/2026, 3:30:45 AM" (en-US) or "2026-10-12 03:30:45" (ISO). */
export function formatDateTime(value: string | number | Date, pref: DateFormatPref): string {
  const d = parse(value)
  if (Number.isNaN(d.getTime())) return '-'
  return pref === ISO ? `${isoDate(d)} ${isoTime(d, true)}` : d.toLocaleString(locale(pref))
}

export function formatDate(value: string | number | Date, pref: DateFormatPref): string {
  const d = parse(value)
  if (Number.isNaN(d.getTime())) return '-'
  return pref === ISO ? isoDate(d) : d.toLocaleDateString(locale(pref))
}

/** Time of day, with or without seconds. */
export function formatTime(value: string | number | Date, pref: DateFormatPref, seconds = true): string {
  const d = parse(value)
  if (Number.isNaN(d.getTime())) return '-'
  if (pref === ISO) return isoTime(d, seconds)
  return d.toLocaleTimeString(locale(pref), seconds ? undefined : { hour: '2-digit', minute: '2-digit' })
}

/** Locale choices: the browser's languages first, then common ones, without duplicates. */
export function localeChoices(browserLanguages: readonly string[]): string[] {
  return [...new Set([...browserLanguages, ...COMMON_LOCALES])].filter((l) => normalizePref(l) === l)
}
