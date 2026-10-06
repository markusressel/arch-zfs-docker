import { describe, expect, it } from 'vitest'
import { appendBounded } from './logBuffer'

describe('appendBounded', () => {
  it('appends below the limit', () => {
    expect(appendBounded(['a'], ['b', 'c'], 5)).toEqual(['a', 'b', 'c'])
  })

  it('drops the oldest lines beyond the limit', () => {
    expect(appendBounded(['a', 'b', 'c'], ['d', 'e'], 4)).toEqual(['b', 'c', 'd', 'e'])
  })

  it('keeps only the tail of an oversized chunk', () => {
    expect(appendBounded(['a'], ['b', 'c', 'd', 'e'], 2)).toEqual(['d', 'e'])
  })

  it('does not mutate its input', () => {
    const lines = ['a']
    appendBounded(lines, ['b'], 5)
    expect(lines).toEqual(['a'])
  })
})
