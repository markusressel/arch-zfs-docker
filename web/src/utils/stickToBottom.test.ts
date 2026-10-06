import { expect, it } from 'vitest'
import { isNearBottom } from './stickToBottom'

it('detects whether the viewport is near the bottom', () => {
  expect(isNearBottom({ scrollTop: 920, scrollHeight: 1500, clientHeight: 500 }, 80)).toBe(true)
  expect(isNearBottom({ scrollTop: 100, scrollHeight: 1500, clientHeight: 500 }, 80)).toBe(false)
  expect(isNearBottom({ scrollTop: 0, scrollHeight: 300, clientHeight: 500 }, 80)).toBe(true)
})
