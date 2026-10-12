import { expect, it } from 'vitest'
import { iniTokens, installCommand, pacmanSnippet } from './pacman'

it('renders a multi-line pacman.conf section', () => {
  expect(pacmanSnippet('myrepo', 'http://host')).toBe(
    '[myrepo]\nSigLevel = Optional TrustAll\nServer = http://host/$repo/$arch',
  )
  expect(pacmanSnippet(undefined, 'http://host')).toContain('[zfslocal]')
})

it('installs the ZFS module matching the kernel variant', () => {
  expect(installCommand('')).toBe('sudo pacman -Sy zfs-linux zfs-utils')
  expect(installCommand('lts')).toBe('sudo pacman -Sy zfs-linux-lts zfs-utils')
})

it('tokenizes pacman.conf lines', () => {
  expect(iniTokens('[zfslocal]')).toEqual([{ text: '[zfslocal]', kind: 'section' }])
  expect(iniTokens('Server = http://host/$repo/$arch')).toEqual([
    { text: 'Server', kind: 'key' },
    { text: ' = ', kind: 'punct' },
    { text: 'http://host/$repo/$arch', kind: 'value' },
  ])
  expect(iniTokens('# a comment = not a key')).toEqual([{ text: '# a comment = not a key', kind: 'comment' }])
  expect(iniTokens('')).toEqual([{ text: '', kind: 'plain' }])
})
