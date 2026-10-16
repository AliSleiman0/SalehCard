import { describe, it, expect } from 'vitest'
import { adaptOrder, codeLines, creditLines, customerName, itemQtyLabel } from './adaptOrder'
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

describe('creditLines', () => {
  const credit = {
    ...makeOrder().items[0],
    title: { en: 'LAKA CHAT', ar: '', tr: '' },
    fulfillmentType: 'account_credit' as const,
    qty: 1,
    amount: 45000,
    price: 11.52,
  }

  it('lists a single credit line', () => {
    const r = creditLines([{ ...credit, playerId: 'AAA111' }])
    expect(r.lines).toEqual([{ title: 'LAKA CHAT', playerId: 'AAA111', qtyLabel: '45,000 units', total: 11.52 }])
    expect(r.distinctIds).toBe(1)
  })

  it('counts two lines for the same account as one', () => {
    const r = creditLines([
      { ...credit, playerId: 'AAA111' },
      { ...credit, playerId: ' AAA111 ' },
    ])
    expect(r.lines).toHaveLength(2)
    expect(r.distinctIds).toBe(1)
  })

  it('counts different accounts', () => {
    const r = creditLines([
      { ...credit, playerId: 'AAA111' },
      { ...credit, playerId: 'BBB222' },
    ])
    expect(r.distinctIds).toBe(2)
  })

  it('falls back to the first captured field when playerId is empty', () => {
    const r = creditLines([
      { ...credit, playerId: '', fields: [{ key: 'field_1', label: { en: 'ID', ar: '', tr: '' }, value: 'ZZ9' }] },
    ])
    expect(r.lines[0].playerId).toBe('ZZ9')
  })

  it('ignores non-credit lines and uses price × qty', () => {
    const r = creditLines([
      makeOrder().items[0],
      { ...credit, amount: undefined, qty: 3, price: 2, playerId: 'P1' },
    ])
    expect(r.lines).toEqual([{ title: 'LAKA CHAT', playerId: 'P1', qtyLabel: 'Qty 3', total: 6 }])
  })
})

describe('itemQtyLabel', () => {
  it('shows the typed amount for a per-unit line', () => {
    expect(itemQtyLabel({ qty: 1, amount: 45000 })).toBe('45,000 units')
  })

  it('shows Qty for a package line', () => {
    expect(itemQtyLabel({ qty: 3 })).toBe('Qty 3')
  })
})

describe('codeLines', () => {
  const code = makeOrder().items[0]
  const credit = { ...code, title: { en: 'LAKA', ar: '', tr: '' }, fulfillmentType: 'account_credit' as const }

  it('groups codes per line for a pure code order (qty 2 + second line)', () => {
    const r = codeLines(
      [
        { ...code, qty: 2, codes: ['A1', 'A2'] },
        { ...code, title: { en: 'Google Play', ar: '', tr: '' }, codes: ['B1'] },
      ],
      'A1',
    )
    expect(r).toEqual([
      { title: 'Steam Wallet', codes: ['A1', 'A2'] },
      { title: 'Google Play', codes: ['B1'] },
    ])
  })

  it('returns the code lines of a mixed order and skips the credit line', () => {
    const r = codeLines([{ ...credit, playerId: 'P1' }, { ...code, codes: ['C1'] }], 'C1')
    expect(r).toEqual([{ title: 'Steam Wallet', codes: ['C1'] }])
    expect(creditLines([{ ...credit, playerId: 'P1' }, { ...code, codes: ['C1'] }]).lines).toHaveLength(1)
  })

  it('falls back to deliveredCode for an old order without items[].codes', () => {
    expect(codeLines([credit, code], 'OLD')).toEqual([{ title: 'Steam Wallet', codes: ['OLD'] }])
  })

  it('is empty for a pure credit order', () => {
    expect(codeLines([{ ...credit, playerId: 'P1' }])).toEqual([])
  })
})
