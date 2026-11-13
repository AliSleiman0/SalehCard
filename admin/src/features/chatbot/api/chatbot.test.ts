import { describe, it, expect } from 'vitest'
import { groupEntries, type ChatEntry } from './chatbot'

const e = (id: string, sortOrder: number, parentId?: string): ChatEntry => ({
  id,
  parentId,
  question: { en: id, ar: '' },
  answer: { en: 'a', ar: '' },
  sortOrder,
  enabled: true,
  createdAt: '2026-11-13T00:00:00Z',
  updatedAt: '2026-11-13T00:00:00Z',
})

describe('groupEntries', () => {
  it('orders top-level questions with their follow-ups, orphans last', () => {
    const g = groupEntries([e('b', 2), e('a', 1), e('a2', 2, 'a'), e('a1', 1, 'a'), e('x', 0, 'gone')])
    expect(g.map((x) => x.top.id)).toEqual(['a', 'b', 'x'])
    expect(g[0].children.map((c) => c.id)).toEqual(['a1', 'a2'])
    expect(g[1].children).toEqual([])
  })
})
