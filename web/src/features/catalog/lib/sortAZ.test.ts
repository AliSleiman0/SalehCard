import { describe, it, expect } from 'vitest'
import { sortAZ } from './sortAZ'

describe('sortAZ', () => {
  it('sorts names naturally and case-insensitively without mutating', () => {
    const input = [{ brand: 'jawaker 400' }, { brand: 'Bulala' }, { brand: 'JAWAKER 150' }, { brand: 'apple' }]
    const out = sortAZ(input)
    expect(out.map((x) => x.brand)).toEqual(['apple', 'Bulala', 'JAWAKER 150', 'jawaker 400'])
    expect(input[0].brand).toBe('jawaker 400')
  })
  it('ignores leading punctuation and puts blank names last', () => {
    const out = sortAZ([{ brand: '' }, { brand: '(Sigma Plus' }, { brand: 'Apple' }])
    expect(out.map((x) => x.brand)).toEqual(['Apple', '(Sigma Plus', ''])
  })
})
