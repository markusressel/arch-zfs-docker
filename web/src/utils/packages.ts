import type { PackageInfo } from '../types/api'

export function filterPackages(packages: PackageInfo[], term: string): PackageInfo[] {
  const q = term.toLowerCase().trim()
  if (!q) return packages
  return packages.filter(
    (p) =>
      p.packageName.toLowerCase().includes(q) ||
      p.version.toLowerCase().includes(q) ||
      (p.kernelVersion ?? '').toLowerCase().includes(q),
  )
}
