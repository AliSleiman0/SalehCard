import { describe, it, expect } from 'vitest'
import { adaptPromo, periodLabel, valueLabel } from './adaptPromo'
import type { AdminPromo } from '../api/promos'

function makePromo(overrides: Partial<AdminPromo> = {}): AdminPromo {
  return {
    id: '6a40134ab53dd98fddb5fef4',
    code: 'WELCOME10',
    type: 'percent',
    value: 10,
    minOrder: 0,
    maxUses: 5000,
    uses: 1840,
    expiresAt: '2026-07-30T00:00:00.000Z',
    active: true,
    status: 'active',
    createdAt: '2026-06-01T00:00:00.000Z',
    ...overrides,
  }
}

describe('valueLabel', () => {
  it('formats by type', () => {
    expect(valueLabel('percent', 10)).toBe('10%')
    expect(valueLabel('fixed', 5)).toBe('$5')
    expect(valueLabel('cashback', 3)).toBe('$3 back')
  })
})

describe('periodLabel', () => {
  it('renders both bounds, one bound, or always', () => {
    expect(periodLabel('2026-06-01T00:00:00Z', '2026-06-30T00:00:00Z')).toBe('Jun 1 – Jun 30')
    expect(periodLabel(undefined, '2026-06-30T00:00:00Z')).toBe('Until Jun 30')
    expect(periodLabel('2026-06-01T00:00:00Z', undefined)).toBe('From Jun 1')
    expect(periodLabel(undefined, undefined)).toBe('Always')
  })
})

describe('adaptPromo', () => {
  it('maps the core fields and computes usage', () => {
    const v = adaptPromo(makePromo())
    expect(v.code).toBe('WELCOME10')
    expect(v.valueLabel).toBe('10%')
    expect(v.used).toBe(1840)
    expect(v.limit).toBe(5000)
    expect(v.unlimited).toBe(false)
    expect(v.usagePct).toBe(37) // 1840/5000 = 36.8 -> 37
    expect(v.status).toBe('active')
  })

  it('treats maxUses 0 as unlimited (0% usage)', () => {
    const v = adaptPromo(makePromo({ maxUses: 0, uses: 96 }))
    expect(v.unlimited).toBe(true)
    expect(v.usagePct).toBe(0)
  })

  it('caps usage at 100%', () => {
    const v = adaptPromo(makePromo({ maxUses: 50, uses: 50, status: 'depleted' }))
    expect(v.usagePct).toBe(100)
    expect(v.status).toBe('depleted')
  })
})
