import { describe, it, expect } from 'vitest'
import { filterCatalog } from './filterCatalog'

const rows = [
  { upstreamId: 'jt-101', name: 'Path to Nowhere — Weekly Arrest Pack', category: 'Path To Nowhere' },
  { upstreamId: 'jt-202', name: 'PUBG 60 UC', category: 'PUBG Mobile' },
  { upstreamId: 'jt-303', name: 'Free Fire 100', category: 'Garena' },
]

describe('filterCatalog', () => {
  it('matches name, category or upstream id, case-insensitively', () => {
    expect(filterCatalog(rows, 'pubg').map((r) => r.upstreamId)).toEqual(['jt-202'])
    expect(filterCatalog(rows, 'garena').map((r) => r.upstreamId)).toEqual(['jt-303'])
    expect(filterCatalog(rows, 'JT-1').map((r) => r.upstreamId)).toEqual(['jt-101'])
  })
  it('returns everything for a blank term', () => {
    expect(filterCatalog(rows, '  ')).toHaveLength(3)
  })
})
