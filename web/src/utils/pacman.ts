import { variantLabel } from './jobs'

export function pacmanSnippet(repoName: string | undefined, origin: string): string {
  return `[${repoName || 'zfslocal'}]\nSigLevel = Optional TrustAll\nServer = ${origin}/$repo/$arch`
}

/** The pacman command installing ZFS for a kernel variant ("" is the standard kernel). */
export const installCommand = (variant: string) => `sudo pacman -Syu zfs-${variantLabel(variant)} zfs-utils`

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

/**
 * Packages that are pinned to the kernel version: the kernel, its headers, the ZFS module and zfs-utils
 * (zfs-linux needs the exact zfs-utils version). Ignoring only the kernel is not enough, as pacman then
 * fails on the zfs-linux package of the repository that needs the new kernel.
 */
export function heldPackages(kernelPkg: string): string[] {
  return [kernelPkg, `${kernelPkg}-headers`, `zfs-${kernelPkg}`, `zfs-${kernelPkg}-headers`, 'zfs-utils']
}

/** A full upgrade except for the kernel and ZFS, for while no package for the new kernel is built. */
export function heldUpdateCommand(kernelPkg: string): string {
  return `sudo pacman -Syu --ignore ${heldPackages(kernelPkg).join(',')}`
}
