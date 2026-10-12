import { variantLabel } from './jobs'

export function pacmanSnippet(repoName: string | undefined, origin: string): string {
  return `[${repoName || 'zfslocal'}]\nSigLevel = Optional TrustAll\nServer = ${origin}/$repo/$arch`
}

/** The pacman command installing ZFS for a kernel variant ("" is the standard kernel). */
export const installCommand = (variant: string) => `sudo pacman -Sy zfs-${variantLabel(variant)} zfs-utils`

export interface IniToken {
  text: string
  kind: 'section' | 'key' | 'punct' | 'value' | 'comment' | 'plain'
}

/** Splits one line of an INI file such as pacman.conf into tokens for syntax highlighting. */
export function iniTokens(line: string): IniToken[] {
  if (/^\s*#/.test(line)) return [{ text: line, kind: 'comment' }]
  if (/^\s*\[.*\]\s*$/.test(line)) return [{ text: line, kind: 'section' }]
  const m = /^(\s*[^=]*?)(\s*=\s*)(.*)$/.exec(line)
  if (!m) return [{ text: line, kind: 'plain' }]
  return [
    { text: m[1], kind: 'key' },
    { text: m[2], kind: 'punct' },
    { text: m[3], kind: 'value' },
  ]
}
