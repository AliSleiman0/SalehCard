import { describe, it, expect } from 'vitest'
import { acceptsQty, estimateCost, pickWarnings, qtyHint } from './supplierPick'
import type { CatalogProductView } from '../api/suppliers'

function row(over: Partial<CatalogProductView>): CatalogProductView {
  return {
    upstreamId: '2',
    name: 'SoulChill',
    category: 'Apps',
    parentId: '',
    price: 0.0018,
    basePrice: 0.0018,
    currency: 'USD',
    available: true,
    productType: 'amount',
    params: ['playerId'],
    qtyMin: 1000,
    qtyMax: 1000000,
    qtyValues: [],
    mapped: false,
    ...over,
  }
}

describe('qtyHint / acceptsQty', () => {
  it('describes ranges and discrete values', () => {
    expect(qtyHint(row({}))).toBe('qty 1,000–1,000,000')
    expect(qtyHint(row({ qtyMin: null, qtyMax: null, qtyValues: ['110', '150'] }))).toBe('qty 110 / 150')
    expect(qtyHint(row({ qtyMin: null, qtyMax: null }))).toBe('')
  })
  it('checks a quantity against the limits', () => {
    expect(acceptsQty(row({}), 1)).toBe(false) // the prod "Qty is Small"
    expect(acceptsQty(row({}), 1000)).toBe(true)
    expect(acceptsQty(row({ qtyMin: null, qtyMax: null, qtyValues: ['110'] }), 110)).toBe(true)
    expect(acceptsQty(row({ qtyMin: null, qtyMax: null }), 7)).toBe(true)
  })
})

describe('pickWarnings', () => {
  const catalog = [row({}), row({ upstreamId: '9', available: false, qtyMin: null, qtyMax: null })]

  it('flags an id that is not in the catalog (the prod "soulchill" link)', () => {
    const w = pickWarnings({ value: 'soulchill', catalog })
    expect(w).toHaveLength(1)
    expect(w[0]).toMatch(/not in this supplier's catalog/)
  })
  it('stays quiet without a catalog or a value', () => {
    expect(pickWarnings({ value: 'soulchill' })).toEqual([])
    expect(pickWarnings({ value: ' ', catalog })).toEqual([])
  })
  it('flags an order quantity the supplier refuses', () => {
    expect(pickWarnings({ value: '2', catalog, qty: 1 })[0]).toMatch(/sends quantity 1/)
    expect(pickWarnings({ value: '2', catalog, qty: 5000 })).toEqual([])
  })
  it('flags an amount range wider than the supplier accepts', () => {
    expect(pickWarnings({ value: '2', catalog, amountRange: { min: 100, max: 5000 } })[0]).toMatch(/narrow/)
    expect(pickWarnings({ value: '2', catalog, amountRange: { min: 1000, max: 5000 } })).toEqual([])
  })
  it('flags supplier parameters the order does not send', () => {
    expect(pickWarnings({ value: '2', catalog, qty: 1000, sends: ['id', 'amount'] })[0]).toMatch(/"playerId"/)
    expect(pickWarnings({ value: '2', catalog, qty: 1000, sends: ['id', 'playerid'] })).toEqual([])
  })
  it('matches imported (slugified) keys and skips unmatchable Arabic prompts', () => {
    const imported = [row({ params: ['Player ID', 'Zone ID', 'رقم الهاتف'] })]
    expect(pickWarnings({ value: '2', catalog: imported, qty: 1000, sends: ['player_id', 'zone_id', 'field3'] })).toEqual([])
    expect(pickWarnings({ value: '2', catalog: imported, qty: 1000, sends: ['player_id'] })[0]).toMatch(/"Zone ID"/)
  })
  it('flags an unavailable product', () => {
    expect(pickWarnings({ value: '9', catalog })[0]).toMatch(/unavailable/)
  })
})

describe('estimateCost', () => {
  it('multiplies the catalog price by the units', () => {
    expect(estimateCost(row({}), 1000)).toBe(1.8)
    expect(estimateCost(undefined, 1000)).toBeNull()
    expect(estimateCost(row({ price: 0 }), 1000)).toBeNull()
  })
})
