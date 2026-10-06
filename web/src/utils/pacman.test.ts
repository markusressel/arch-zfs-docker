import { expect, it } from 'vitest'
import { pacmanSnippet } from './pacman'

it('renders a multi-line pacman.conf section', () => {
  expect(pacmanSnippet('myrepo', 'http://host')).toBe(
    '[myrepo]\nSigLevel = Optional TrustAll\nServer = http://host/$repo/$arch',
  )
  expect(pacmanSnippet(undefined, 'http://host')).toContain('[zfslocal]')
})
