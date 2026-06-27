import { describe, it, expect } from 'vitest'
import { adaptReseller } from './adaptReseller'
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
