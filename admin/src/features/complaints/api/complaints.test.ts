import { describe, it, expect } from 'vitest'
import { contactLabel, whatsappLink } from './complaints'

describe('whatsappLink', () => {
  it('keeps only digits', () => {
    expect(whatsappLink('+961 78 991 778')).toBe('https://wa.me/96178991778')
  })
  it('is empty without a phone', () => {
    expect(whatsappLink(undefined)).toBe('')
    expect(whatsappLink('n/a')).toBe('')
  })
})

describe('contactLabel', () => {
  it('prefers name, then email, then phone, then a short id', () => {
    expect(contactLabel({ id: 'abcdef1234567890', name: 'Ali', email: 'a@x' })).toBe('Ali')
    expect(contactLabel({ id: 'abcdef1234567890', email: 'a@x' })).toBe('a@x')
    expect(contactLabel({ id: 'abcdef1234567890', phone: '+961' })).toBe('+961')
    expect(contactLabel({ id: 'abcdef1234567890' })).toBe('34567890')
    expect(contactLabel(undefined)).toBe('—')
  })
})
