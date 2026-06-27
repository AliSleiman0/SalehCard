import { describe, it, expect } from 'vitest'
import { adaptOrder, customerName } from './adaptOrder'
import type { AdminOrder } from '../api/orders'

function makeOrder(overrides: Partial<AdminOrder> = {}): AdminOrder {
  return {
    id: '6a3fcf7d6f3ba8189640f419',
    userId: 'u1',
    items: [
      {
        productId: 'p1',
        variantId: 'v1',
        title: { en: 'Steam Wallet', ar: '', tr: '' },
        category: 'giftcards',
        qty: 1,
        price: 50,
        fulfillmentType: 'code',
      },
    ],
    subtotal: 50,
    total: 50,
    currency: 'USD',
    paymentMethod: 'card',
    status: 'completed',
    fulfillment: { statusTimeline: [] },
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    customer: { id: 'u1', email: 'customer@salehcard.local' },
    ...overrides,
  }
}

describe('adaptOrder', () => {
  it('maps the core fields', () => {
    const v = adaptOrder(makeOrder())
    expect(v.id).toBe('6a3fcf7d6f3ba8189640f419')
    expect(v.product).toBe('Steam Wallet')
    expect(v.qty).toBe(1)
    expect(v.amount).toBe(50)
    expect(v.cur).toBe('USD')
    expect(v.status).toBe('completed')
  })

  it('maps enum values: card → visa, code → code, account_credit → credit', () => {
    expect(adaptOrder(makeOrder({ paymentMethod: 'card' })).pay).toBe('visa')
    expect(adaptOrder(makeOrder({ paymentMethod: 'wallet' })).pay).toBe('wallet')
    expect(adaptOrder(makeOrder({ paymentMethod: 'usdt' })).pay).toBe('usdt')

    const credit = adaptOrder(
      makeOrder({ items: [{ ...makeOrder().items[0], fulfillmentType: 'account_credit' }] })
    )
    expect(credit.ff).toBe('credit')
    const transfer = adaptOrder(
      makeOrder({ items: [{ ...makeOrder().items[0], fulfillmentType: 'transfer' }] })
    )
    expect(transfer.ff).toBe('transfer')
  })

  it('derives a display name from the email local-part', () => {
    expect(adaptOrder(makeOrder()).customer).toBe('customer')
    expect(adaptOrder(makeOrder()).email).toBe('customer@salehcard.local')
  })

  it('falls back to phone, then Unknown, when email is absent', () => {
    expect(customerName({ id: 'x', email: '', phone: '+96170000000' })).toBe('+96170000000')
    expect(customerName(null)).toBe('Unknown')
  })

  it('summarises multi-item orders with "+N more" and summed qty', () => {
    const base = makeOrder().items[0]
    const v = adaptOrder(
      makeOrder({
        items: [
          { ...base, qty: 2 },
          { ...base, title: { en: 'PUBG UC', ar: '', tr: '' }, qty: 3 },
        ],
      })
    )
    expect(v.product).toBe('Steam Wallet +1 more')
    expect(v.qty).toBe(5)
  })

  it('handles an empty item list defensively', () => {
    const v = adaptOrder(makeOrder({ items: [] }))
    expect(v.product).toBe('—')
    expect(v.qty).toBe(0)
    expect(v.ff).toBe('code')
  })
})
