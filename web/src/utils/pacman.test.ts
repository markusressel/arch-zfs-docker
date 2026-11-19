import { describe, expect, it } from 'vitest'
import { heldUpdateCommand, iniTokens, installCommand, pacmanSnippet } from './pacman'

it('renders a multi-line pacman.conf section', () => {
  expect(pacmanSnippet('myrepo', 'http://host')).toBe(
    '[myrepo]\nSigLevel = Optional TrustAll\nServer = http://host/$repo/$arch',
  )
  expect(pacmanSnippet(undefined, 'http://host')).toContain('[zfslocal]')
})

it('installs the ZFS module matching the kernel variant', () => {
  expect(installCommand('')).toBe('sudo pacman -Syu zfs-linux zfs-utils')
  expect(installCommand('lts')).toBe('sudo pacman -Syu zfs-linux-lts zfs-utils')
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

describe('heldUpdateCommand', () => {
  it('ignores the kernel, its headers, the ZFS module and zfs-utils', () => {
    expect(heldUpdateCommand('linux')).toBe('sudo pacman -Syu --ignore linux,linux-headers,zfs-linux,zfs-linux-headers,zfs-utils')
    expect(heldUpdateCommand('linux-lts')).toBe(
      'sudo pacman -Syu --ignore linux-lts,linux-lts-headers,zfs-linux-lts,zfs-linux-lts-headers,zfs-utils',
    )
  })
})
