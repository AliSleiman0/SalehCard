import { describe, it, expect, afterEach } from 'vitest'
import i18n from '@/i18n/config'
import { money, stockLevel, isRTL, formatDateTime, formatDate } from './utils'

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

describe('dates follow the admin language', () => {
  afterEach(() => i18n.changeLanguage('en'))
  const at = new Date(2026, 10, 12, 15, 5)

  it('keeps the English shape', async () => {
    await i18n.changeLanguage('en')
    expect(formatDateTime(at)).toBe('Nov 12, 2026 · 3:05 PM')
    expect(formatDate(at)).toBe('Nov 12, 2026')
  })

  it('writes Arabic month names with Western digits', async () => {
    await i18n.changeLanguage('ar')
    const s = formatDateTime(at)
    expect(s).toContain('نوفمبر')
    expect(s).toContain('2026')
    expect(s).not.toMatch(/[٠-٩]/)
  })
})
