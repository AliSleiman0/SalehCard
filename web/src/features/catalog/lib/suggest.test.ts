import { describe, it, expect } from 'vitest'
import { buildSuggestions, highlight } from './suggest'
import type { Product } from '@/types'

const p = (id: string, en: string, ar = ''): Product =>
  ({ id, title: { en, ar, tr: '' }, images: [], thumbnail: '' }) as unknown as Product

describe('buildSuggestions', () => {
  const list = [p('1', 'Soul Chill'), p('2', 'PUBG Mobile'), p('3', 'Soul Star 10000'), p('4', 'Yalla Soul'), p('5', 'Soul Star 900')]
  it('matches any part of the name; prefix matches first, then A to Z', () => {
    expect(buildSuggestions(list, 'sou', 'en').map((s) => s.name)).toEqual([
      'Soul Chill',
      'Soul Star 900',
      'Soul Star 10000',
      'Yalla Soul',
    ])
  })
  it('is empty for a blank term and respects the cap', () => {
    expect(buildSuggestions(list, '  ', 'en')).toEqual([])
    expect(buildSuggestions(list, 'o', 'en', 2)).toHaveLength(2)
  })
  it('uses the locale name with an English fallback', () => {
    expect(buildSuggestions([p('9', 'Jawaker', 'جواكر')], 'جوا', 'ar')[0].name).toBe('جواكر')
    expect(buildSuggestions([p('9', 'Jawaker')], 'jaw', 'ar')[0].name).toBe('Jawaker')
  })
})

describe('highlight', () => {
  it('splits around the first match', () => {
    expect(highlight('Soul Star', 'STA')).toEqual(['Soul ', 'Sta', 'r'])
    expect(highlight('Soul', 'x')).toEqual(['Soul', '', ''])
  })
})
