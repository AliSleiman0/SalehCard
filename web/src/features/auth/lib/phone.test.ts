import { describe, it, expect } from 'vitest'
import { toE164Lebanon, isPlausibleLebanonPhone, phoneDigits } from './phone'

describe('toE164Lebanon', () => {
  it('prepends +961 to bare digits', () => {
    expect(toE164Lebanon('70123456')).toBe('+96170123456')
  })
  it('drops a single leading zero', () => {
    expect(toE164Lebanon('070123456')).toBe('+96170123456')
  })
  it('strips spaces and punctuation', () => {
    expect(toE164Lebanon('70 123 456')).toBe('+96170123456')
    // local "03" mobile prefix loses its leading zero in E.164
    expect(toE164Lebanon('03-123-456')).toBe('+9613123456')
  })
})

describe('phoneDigits', () => {
  it('keeps only digits', () => {
    expect(phoneDigits('+961 70-123')).toBe('96170123')
  })
})

describe('isPlausibleLebanonPhone', () => {
  it('accepts >= 7 digits after dropping a leading zero', () => {
    expect(isPlausibleLebanonPhone('70123456')).toBe(true)
    expect(isPlausibleLebanonPhone('0 70 123 45')).toBe(true)
  })
  it('rejects short input', () => {
    expect(isPlausibleLebanonPhone('12345')).toBe(false)
    expect(isPlausibleLebanonPhone('')).toBe(false)
  })
})
