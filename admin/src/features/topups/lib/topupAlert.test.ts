import { describe, it, expect } from 'vitest'
import { isNewTopUp, titleWithCount } from './topupAlert'

describe('isNewTopUp', () => {
  it('seeds silently on the first observation', () => {
    expect(isNewTopUp(undefined, 3)).toBe(false)
  })
  it('alerts only when the count rises', () => {
    expect(isNewTopUp(2, 3)).toBe(true)
    expect(isNewTopUp(3, 3)).toBe(false)
    expect(isNewTopUp(3, 2)).toBe(false) // an admin approved one
    expect(isNewTopUp(3, undefined)).toBe(false)
  })
})

describe('titleWithCount', () => {
  it('prefixes and replaces the pending count', () => {
    expect(titleWithCount('FlashCash Admin', 2)).toBe('(2) FlashCash Admin')
    expect(titleWithCount('(2) FlashCash Admin', 5)).toBe('(5) FlashCash Admin')
    expect(titleWithCount('(2) FlashCash Admin', 0)).toBe('FlashCash Admin')
  })
})
