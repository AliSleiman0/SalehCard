import { describe, it, expect } from 'vitest'
import { money, stockLevel, isRTL } from './utils'

describe('money', () => {
  it('formats USD with a dollar sign', () => {
    expect(money(1234.5, 'USD')).toBe('$1,234.50')
  })
  it('formats TRY with a lira sign', () => {
    expect(money(1234.5, 'TRY')).toBe('₺1,234.50')
  })
  it('renders negatives with a minus sign', () => {
    expect(money(-9.99)).toBe('−$9.99')
  })
})

describe('stockLevel', () => {
  it('is lo when empty or far below threshold', () => {
    expect(stockLevel(0, 100)).toBe('lo')
    expect(stockLevel(30, 100)).toBe('lo')
  })
  it('is mid when below threshold', () => {
    expect(stockLevel(70, 100)).toBe('mid')
  })
  it('is hi when at or above threshold', () => {
    expect(stockLevel(120, 100)).toBe('hi')
  })
})

describe('isRTL', () => {
  it('is true only for Arabic', () => {
    expect(isRTL('ar')).toBe(true)
    expect(isRTL('en')).toBe(false)
    expect(isRTL('tr')).toBe(false)
  })
})
