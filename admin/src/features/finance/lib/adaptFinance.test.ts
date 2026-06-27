import { describe, it, expect } from 'vitest'
import { adaptTx, payKey } from './adaptFinance'
import type { AdminTx } from '../api/finance'

function makeTx(overrides: Partial<AdminTx> = {}): AdminTx {
  return {
    id: '6a3ff4eec34d4f13cee5c279',
    userId: '6a3fe3ff45c3d79a75de155c',
    type: 'topup',
    amount: 50,
    balanceAfter: 100,
    method: 'card',
    ref: 'seed:topup-card',
    createdAt: '2026-06-20T10:00:00.000Z',
    user: { name: 'gamehub.store', role: 'reseller' },
    ...overrides,
  }
}

describe('payKey', () => {
  it('maps card to the visa PayChip key', () => expect(payKey('card')).toBe('visa'))
  it('blanks admin (manual adjustment has no chip)', () => expect(payKey('admin')).toBe(''))
  it('passes wallet and usdt through', () => {
    expect(payKey('wallet')).toBe('wallet')
    expect(payKey('usdt')).toBe('usdt')
  })
})

describe('adaptTx', () => {
  it('maps the core fields and resolves the method to a PayChip key', () => {
    const v = adaptTx(makeTx())
    expect(v.id).toBe('6a3ff4eec34d4f13cee5c279')
    expect(v.user).toBe('gamehub.store')
    expect(v.role).toBe('reseller')
    expect(v.type).toBe('topup')
    expect(v.amount).toBe(50)
    expect(v.method).toBe('visa')
  })

  it('blanks the method for a manual adjustment', () => {
    expect(adaptTx(makeTx({ type: 'adjustment', method: 'admin' })).method).toBe('')
  })

  it('falls back to Unknown when the owning user is missing (orphaned row)', () => {
    const v = adaptTx(makeTx({ user: undefined as unknown as AdminTx['user'] }))
    expect(v.user).toBe('Unknown')
    expect(v.role).toBe('')
  })
})
