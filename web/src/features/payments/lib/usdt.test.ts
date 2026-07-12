import { describe, it, expect } from 'vitest'
import { formatUsdtAmount } from './usdt'

describe('formatUsdtAmount', () => {
  it('keeps at least 2 decimals', () => {
    expect(formatUsdtAmount(10)).toBe('10.00')
    expect(formatUsdtAmount(24.99)).toBe('24.99')
  })
  it('preserves the salted sub-cent identity digits', () => {
    expect(formatUsdtAmount(24.990001)).toBe('24.990001')
    expect(formatUsdtAmount(1.2345)).toBe('1.2345')
    expect(formatUsdtAmount(25.004913)).toBe('25.004913')
  })
  it('trims trailing zeros down to 2 decimals', () => {
    expect(formatUsdtAmount(5.5)).toBe('5.50')
    expect(formatUsdtAmount(5.5000)).toBe('5.50')
    expect(formatUsdtAmount(1.234500)).toBe('1.2345')
  })
})
