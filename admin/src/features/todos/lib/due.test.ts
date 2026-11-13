import { describe, it, expect } from 'vitest'
import { dateInputToIso, dueState } from './due'

describe('dueState', () => {
  const now = new Date(2026, 10, 13, 15, 0)
  it('classifies by local calendar day', () => {
    expect(dueState(undefined, now)).toBe('none')
    expect(dueState(new Date(2026, 10, 12, 23, 0).toISOString(), now)).toBe('overdue')
    expect(dueState(new Date(2026, 10, 13, 8, 0).toISOString(), now)).toBe('today')
    expect(dueState(new Date(2026, 10, 14, 1, 0).toISOString(), now)).toBe('upcoming')
  })
})

describe('dateInputToIso', () => {
  it('keeps the picked calendar day', () => {
    const iso = dateInputToIso('2026-11-20')!
    const d = new Date(iso)
    expect([d.getFullYear(), d.getMonth() + 1, d.getDate()]).toEqual([2026, 11, 20])
    expect(dateInputToIso('')).toBeUndefined()
  })
})
