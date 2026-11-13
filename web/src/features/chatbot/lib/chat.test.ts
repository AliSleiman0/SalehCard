import { describe, it, expect } from 'vitest'
import { filterEntries, followUps, text, topLevel, whatsappLink } from './chat'
import type { ChatEntry } from '../api/chatbot'

const e = (id: string, sortOrder: number, over: Partial<ChatEntry> = {}): ChatEntry => ({
  id,
  question: { en: id, ar: '' },
  answer: { en: 'a', ar: '' },
  sortOrder,
  ...over,
})

describe('chat helpers', () => {
  const all = [
    e('Top up wallet', 2, { keywords: ['wallet', 'شحن'] }),
    e('Delivery time', 1),
    e('Approval time', 1, { parentId: 'Top up wallet' }),
  ]

  it('orders top-level questions and their follow-ups', () => {
    expect(topLevel(all).map((x) => x.id)).toEqual(['Delivery time', 'Top up wallet'])
    expect(followUps(all, 'Top up wallet').map((x) => x.id)).toEqual(['Approval time'])
  })

  it('filters by question text and keywords (any language)', () => {
    expect(filterEntries(all, 'deliv').map((x) => x.id)).toEqual(['Delivery time'])
    expect(filterEntries(all, 'شحن').map((x) => x.id)).toEqual(['Top up wallet'])
    expect(filterEntries(all, '  ')).toEqual([])
  })

  it('falls back between languages', () => {
    expect(text({ en: '', ar: 'مرحبا' }, 'en')).toBe('مرحبا')
    expect(text({ en: 'Hi', ar: '' }, 'ar')).toBe('Hi')
  })

  it('builds wa.me links from free-typed numbers', () => {
    expect(whatsappLink('+961 70 123 456')).toBe('https://wa.me/96170123456')
    expect(whatsappLink('00961 70 123456')).toBe('https://wa.me/96170123456')
    expect(whatsappLink('abc')).toBeNull()
  })
})
