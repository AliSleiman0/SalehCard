import { describe, it, expect } from 'vitest'
import { adaptReview } from './adaptReview'
import type { AdminReview } from '../api/reviews'

function makeReview(overrides: Partial<AdminReview> = {}): AdminReview {
  return {
    id: '6a40134ab53dd98fddb5fef4',
    productId: '6a40134ab53dd98fddb5aaaa',
    productName: 'PUBG Mobile UC',
    art: 'https://cdn.salehcard.com/images/pubg-uc.png',
    userId: '6a40134ab53dd98fddb5bbbb',
    userName: 'omar.haddad',
    rating: 5,
    body: 'Instant delivery.',
    verifiedPurchase: true,
    status: 'pending',
    createdAt: new Date().toISOString(),
    ...overrides,
  }
}

describe('adaptReview', () => {
  it('maps the core fields', () => {
    const v = adaptReview(makeReview())
    expect(v.product).toBe('PUBG Mobile UC')
    expect(v.user).toBe('omar.haddad')
    expect(v.rating).toBe(5)
    expect(v.status).toBe('pending')
    expect(v.verified).toBe(true)
    expect(v.date).toBeTruthy()
  })

  it('flags 1-star reviews as possible spam', () => {
    expect(adaptReview(makeReview({ rating: 1 })).flagged).toBe(true)
    expect(adaptReview(makeReview({ rating: 2 })).flagged).toBe(false)
  })

  it('falls back when product/user did not join', () => {
    const v = adaptReview(makeReview({ productName: '', userName: '' }))
    expect(v.product).toBe('Unknown product')
    expect(v.user).toBe('Unknown')
  })
})
