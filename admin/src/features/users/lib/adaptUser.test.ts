import { describe, it, expect } from 'vitest'
import { adaptUser, userName, joinedLabel } from './adaptUser'
import type { AdminUser } from '../api/users'

function makeUser(overrides: Partial<AdminUser> = {}): AdminUser {
  return {
    id: '6a3fcf7d6f3ba8189640f419',
    email: 'sara.nasser@salehcard.local',
    role: 'customer',
    status: 'active',
    locale: 'en',
    savedPlayerIds: [],
    walletBalance: 58.75,
    loyaltyPoints: 90,
    createdAt: '2025-03-12T08:00:00.000Z',
    updatedAt: '2025-03-12T08:00:00.000Z',
    orders: 38,
    spent: 1240,
    ...overrides,
  }
}

describe('adaptUser', () => {
  it('maps the core fields', () => {
    const v = adaptUser(makeUser())
    expect(v.id).toBe('6a3fcf7d6f3ba8189640f419')
    expect(v.email).toBe('sara.nasser@salehcard.local')
    expect(v.role).toBe('customer')
    expect(v.status).toBe('active')
    expect(v.balance).toBe(58.75)
    expect(v.cur).toBe('USD')
    expect(v.orders).toBe(38)
    expect(v.spent).toBe(1240)
    expect(v.loyalty).toBe(90)
  })

  it('derives a display name from the email local-part', () => {
    expect(adaptUser(makeUser()).name).toBe('sara.nasser')
  })

  it('falls back to phone, then Unknown, when email is absent', () => {
    expect(userName({ email: '', phone: '+96170123456' })).toBe('+96170123456')
    expect(userName({ email: '', phone: undefined })).toBe('Unknown')
  })

  it('treats an empty status as active', () => {
    expect(adaptUser(makeUser({ status: '' as AdminUser['status'] })).status).toBe('active')
  })

  it('formats the join date as "Mon YYYY"', () => {
    expect(joinedLabel('2025-03-12T08:00:00.000Z')).toBe('Mar 2025')
    expect(joinedLabel('not-a-date')).toBe('—')
  })
})
