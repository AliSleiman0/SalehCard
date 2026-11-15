import { describe, it, expect } from 'vitest'
import {
  addLink,
  blockCost,
  fmtCost,
  linkIssues,
  makeActive,
  marginPct,
  moveLink,
  removeLink,
  setEnabled,
  toLinkRows,
  toLinksInput,
} from './supplierLinks'

const saved = toLinkRows(
  [
    { id: 'a', provider: 10, upstreamId: '3258', enabled: true },
    { id: 'b', provider: 11, upstreamId: '4410', enabled: true },
  ],
  'a',
)

describe('supplier links table', () => {
  it('reads the active link from the product', () => {
    expect(saved.map((r) => r.active)).toEqual([true, false])
  })

  it('first added link becomes active; a duplicate is ignored', () => {
    const one = addLink([], 10, '3258', 'SOUL STAR')
    expect(one).toHaveLength(1)
    expect(one[0].active).toBe(true)
    const two = addLink(one, 11, '4410')
    expect(two[1].active).toBe(false)
    expect(addLink(two, 10, '3258')).toBe(two)
  })

  it('reorders, makes active and refuses to disable the active link', () => {
    const moved = moveLink(saved, 1, -1)
    expect(moved.map((r) => r.id)).toEqual(['b', 'a'])
    expect(moveLink(saved, 0, -1)).toBe(saved)
    const act = makeActive(setEnabled(saved, 1, false), 1)
    expect(act.map((r) => r.active)).toEqual([false, true])
    expect(act[1].enabled).toBe(true)
    expect(setEnabled(saved, 0, false)).toBe(saved)
  })

  it('removing the active link hands "active" to the next enabled one', () => {
    const rows = setEnabled(addLink(saved, 12, '77'), 1, false)
    const left = removeLink(rows, 0)
    expect(left.map((r) => r.upstreamId)).toEqual(['4410', '77'])
    expect(left.map((r) => r.active)).toEqual([false, true])
    expect(removeLink(removeLink(saved, 0), 0)).toEqual([])
  })

  it('builds the save payload (ids only for saved rows)', () => {
    const rows = addLink(saved, 12, '77')
    expect(toLinksInput(rows, 3)).toEqual({
      rev: 3,
      links: [
        { id: 'a', provider: 10, upstreamId: '3258', enabled: true, active: true },
        { id: 'b', provider: 11, upstreamId: '4410', enabled: true, active: false },
        { provider: 12, upstreamId: '77', enabled: true, active: false },
      ],
    })
  })

  it('flags an empty table on a supplier product and a missing active row', () => {
    expect(linkIssues([], true)).toHaveLength(1)
    expect(linkIssues([], false)).toEqual([])
    expect(linkIssues(saved.map((r) => ({ ...r, active: false })), true)).toEqual(['Choose exactly one active supplier.'])
  })
})

describe('cost and margin', () => {
  it('prices one selling block from the supplier unit price', () => {
    const q = { provider: 10, upstreamId: '3258', status: 'ok' as const, price: 0.0001096153, currency: 'USD', available: true }
    const cost = blockCost(q, 10000)!
    expect(cost).toBeCloseTo(1.096, 3)
    expect(marginPct(1.23, cost)).toBeCloseTo(10.9, 1)
    expect(blockCost({ ...q, status: 'unknown' }, 10000)).toBeNull()
    expect(blockCost({ ...q, price: 2 }, 0)).toBe(2)
  })
})

describe('fmtCost', () => {
  it('formats a credit balance with the sign before the dollar', () => {
    expect(fmtCost(-1861.19)).toBe('-$1,861.19')
    expect(fmtCost(0.47)).toBe('$0.47')
    expect(fmtCost(0.000123)).toBe('$0.00012')
    expect(fmtCost(1234.5, 'LBP')).toBe('1,234.5 LBP')
  })
})
