import { describe, it, expect } from 'vitest'
import { adaptOrder, codeLines } from './adaptOrder'
import type { Order, OrderItem } from '@/types'

function item(over: Partial<OrderItem>): OrderItem {
  return {
    productId: 'p1',
    variantId: 'v1',
    title: { en: 'Google Play 25', ar: '', tr: '' },
    denomination: '$25',
    category: 'gift',
    qty: 1,
    price: 25,
    fulfillmentType: 'code',
    ...over,
  }
}

function order(items: OrderItem[], over: Partial<Order> = {}): Order {
  return {
    id: 'o1',
    userId: 'u1',
    items,
    total: 50,
    currency: 'USD',
    paymentMethod: 'wallet',
    status: 'completed',
    fulfillment: { statusTimeline: [] },
    createdAt: '2026-10-16T10:00:00Z',
    updatedAt: '2026-10-16T10:00:00Z',
    ...over,
  } as Order
}

describe('adaptOrder code lines', () => {
  it('lists every code of a pure code order, per line (qty 2 + second line)', () => {
    const o = order(
      [
        item({ qty: 2, codes: ['AAA', 'BBB'] }),
        item({ productId: 'p2', title: { en: 'Steam 10', ar: '', tr: '' }, codes: ['CCC'] }),
      ],
      { fulfillment: { deliveredCode: 'AAA', statusTimeline: [] } },
    )
    const v = adaptOrder(o, 'en')
    expect(v.codeLines).toEqual([
      { title: 'Google Play 25', codes: ['AAA', 'BBB'] },
      { title: 'Steam 10', codes: ['CCC'] },
    ])
    expect(v.code).toBe('AAA')
    expect(v.manualKind).toBeUndefined()
    expect(v.fulfill).toBe('code')
  })

  it('keeps codes and the credit line of a mixed processing order', () => {
    const o = order(
      [
        item({
          title: { en: 'LAKA', ar: '', tr: '' },
          fulfillmentType: 'account_credit',
          denomination: '45,000',
          playerId: 'P-1',
        }),
        item({ codes: ['GP-1'] }),
      ],
      { status: 'processing', fulfillment: { deliveredCode: 'GP-1', statusTimeline: [] } },
    )
    const v = adaptOrder(o, 'en')
    expect(v.status).toBe('processing')
    expect(v.fulfill).toBe('credit')
    expect(v.codeLines).toEqual([{ title: 'Google Play 25', codes: ['GP-1'] }])
    expect(v.manualKind).toBe('credit')
    expect(v.account).toBe('P-1')
    expect(v.amount).toBe('45,000')
  })

  it('takes the credit details from the first manual line even when a code line is first', () => {
    const o = order(
      [item({ codes: ['GP-1'] }), item({ fulfillmentType: 'account_credit', denomination: '100', playerId: 'X9' })],
      { status: 'processing' },
    )
    const v = adaptOrder(o, 'en')
    expect(v.fulfill).toBe('code')
    expect(v.manualKind).toBe('credit')
    expect(v.account).toBe('X9')
  })

  it('falls back to fulfillment.deliveredCode for old orders without items[].codes', () => {
    const o = order([item({})], { fulfillment: { deliveredCode: 'OLD-1', statusTimeline: [] } })
    expect(codeLines(o, 'en')).toEqual([{ title: 'Google Play 25', codes: ['OLD-1'] }])
    expect(adaptOrder(o, 'en').code).toBe('OLD-1')
  })

  it('has no code lines for a pure credit order', () => {
    const o = order([item({ fulfillmentType: 'account_credit', denomination: '1,000', playerId: 'P2' })], {
      fulfillment: { creditedToId: 'P2', statusTimeline: [] },
    })
    const v = adaptOrder(o, 'en')
    expect(v.codeLines).toEqual([])
    expect(v.fulfill).toBe('credit')
    expect(v.manualKind).toBe('credit')
    expect(v.account).toBe('P2')
    expect(v.code).toBeUndefined()
  })
})
