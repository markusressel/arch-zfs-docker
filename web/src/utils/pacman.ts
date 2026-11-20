import { variantLabel } from './jobs'

/** The pacman.conf section of the repository; signed repositories require valid signatures. */
export function pacmanSnippet(repoName: string | undefined, origin: string, signed = false): string {
  const sigLevel = signed ? 'Required' : 'Optional TrustAll'
  return `[${repoName || 'zfslocal'}]\nSigLevel = ${sigLevel}\nServer = ${origin}/$repo/$arch`
}

/** Commands that download the signing key and make pacman trust it. */
export function trustKeyCommands(origin: string, keyPath: string, fingerprint: string): string {
  const file = keyPath.split('/').pop() || 'repository.asc'
  return [
    `curl -fsSL -o ${file} ${origin}${keyPath}`,
    `sudo pacman-key --add ${file}`,
    `sudo pacman-key --lsign-key ${fingerprint}`,
  ].join('\n')
}

/** A fingerprint in groups of four characters, as gpg prints it. */
export const formatFingerprint = (fpr: string) => fpr.match(/.{1,4}/g)?.join(' ') ?? fpr

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
