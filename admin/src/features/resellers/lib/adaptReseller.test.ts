import { describe, it, expect } from 'vitest'
import type { Variant } from '@/types'
import { adaptReseller, effectivePrice } from './adaptReseller'
import type { AdminReseller } from '../api/resellers'

function makeReseller(overrides: Partial<AdminReseller> = {}): AdminReseller {
  return {
    id: '6a3fcf7d6f3ba8189640f419',
    email: 'gamehub.store@salehcard.local',
    status: 'active',
    resellerTier: 'Gold',
    margin: 12,
    walletBalance: 4820,
    loyaltyPoints: 5400,
    savedPlayerIds: [],
    createdAt: '2025-03-12T08:00:00.000Z',
    updatedAt: '2025-03-12T08:00:00.000Z',
    orders: 612,
    volume: 28400,
    ...overrides,
  }
}

describe('adaptReseller', () => {
  it('maps the core fields', () => {
    const v = adaptReseller(makeReseller())
    expect(v.id).toBe('6a3fcf7d6f3ba8189640f419')
    expect(v.email).toBe('gamehub.store@salehcard.local')
    expect(v.tier).toBe('Gold')
    expect(v.status).toBe('active')
    expect(v.balance).toBe(4820)
    expect(v.cur).toBe('USD')
    expect(v.margin).toBe(12)
    expect(v.orders).toBe(612)
    expect(v.vol).toBe(28400)
  })

  it('derives a display name from the email local-part', () => {
    expect(adaptReseller(makeReseller()).name).toBe('gamehub.store')
  })

  it('treats an empty status as active and a missing tier as unassigned', () => {
    const v = adaptReseller(
      makeReseller({ status: '' as AdminReseller['status'], resellerTier: undefined }),
    )
    expect(v.status).toBe('active')
    expect(v.tier).toBe('')
  })

  it('formats the join date as "Mon YYYY"', () => {
    expect(adaptReseller(makeReseller()).joined).toBe('Mar 2025')
  })
})

// Mirrors the backend rule (product.ResellerUnitPrice) — the cases below match
// api/internal/modules/product/pricing_test.go.
describe('effectivePrice', () => {
  const variant = (price: number, resellerPrice?: number): Variant => ({
    id: 'v1',
    denomination: 'Default',
    price,
    resellerPrice,
  })

  it('returns retail with no inputs', () => {
    expect(effectivePrice(variant(100), 0)).toBe(100)
  })

  it('applies the tier margin', () => {
    expect(effectivePrice(variant(100), 12)).toBeCloseTo(88)
  })

  it('ignores out-of-range margins', () => {
    expect(effectivePrice(variant(100), 100)).toBe(100)
    expect(effectivePrice(variant(100), -5)).toBe(100)
  })

  it('lets the global override win when lowest', () => {
    expect(effectivePrice(variant(100, 80), 12)).toBe(80)
  })

  it('keeps the margin price when the global override is higher', () => {
    expect(effectivePrice(variant(100, 95), 12)).toBeCloseTo(88)
  })

  it('lets the custom price win when lowest', () => {
    expect(effectivePrice(variant(100, 80), 12, 75)).toBe(75)
  })

  it('keeps the lower layer when the custom price is higher', () => {
    expect(effectivePrice(variant(100), 12, 90)).toBeCloseTo(88)
  })

  it('never exceeds retail', () => {
    expect(effectivePrice(variant(100, 150), 0, 120)).toBe(100)
  })
})
