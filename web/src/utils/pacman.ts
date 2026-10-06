export function pacmanSnippet(repoName: string | undefined, origin: string): string {
  return `[${repoName || 'zfslocal'}]\nSigLevel = Optional TrustAll\nServer = ${origin}/$repo/$arch`
}
