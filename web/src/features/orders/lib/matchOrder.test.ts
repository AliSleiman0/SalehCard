import { describe, it, expect } from 'vitest'
import { matchesOrder, normalizeOrderQuery } from './matchOrder'
import { orderNumber } from '@/lib/orderNumber'
import type { Order, OrderItem } from '@/types'

function item(over: Partial<OrderItem> = {}): OrderItem {
  return {
    productId: 'p1',
    variantId: 'v1',
    title: { en: 'PUBG Mobile 60 UC', ar: 'ببجي موبايل ٦٠ شدة', tr: '' },
    denomination: '60 UC',
    category: 'game',
    qty: 1,
    price: 1,
    fulfillmentType: 'account_credit',
    ...over,
  }
}

function order(over: Partial<Order> = {}, items: OrderItem[] = [item()]): Order {
  return {
    id: '6731f0c2ab45de67a3f9c1d2',
    userId: 'u1',
    items,
    subtotal: 1,
    total: 1,
    currency: 'USD',
    paymentMethod: 'wallet',
    status: 'completed',
    fulfillment: { statusTimeline: [] },
    createdAt: '2026-11-15T10:00:00Z',
    updatedAt: '2026-11-15T10:00:00Z',
    ...over,
  }
}

describe('normalizeOrderQuery', () => {
  it('trims, lower-cases and drops a leading #', () => {
    expect(normalizeOrderQuery('  #A3F9 ')).toBe('a3f9')
    expect(normalizeOrderQuery('##x')).toBe('x')
    expect(normalizeOrderQuery('   ')).toBe('')
  })
})

describe('matchesOrder', () => {
  const o = order()

  it('matches everything on an empty query', () => {
    expect(matchesOrder(o, '')).toBe(true)
    expect(matchesOrder(o, ' # ')).toBe(true)
  })

  it('matches the full order id and a prefix of it', () => {
    expect(matchesOrder(o, '6731f0c2ab45de67a3f9c1d2')).toBe(true)
    expect(matchesOrder(o, '6731F0')).toBe(true)
  })

  it('matches the order number shown on the row, with or without #, and a prefix of it', () => {
    expect(orderNumber(o.id)).toBe('#A3F9C1D2')
    expect(matchesOrder(o, '#A3F9C1D2')).toBe(true)
    expect(matchesOrder(o, 'a3f9c1d2')).toBe(true)
    expect(matchesOrder(o, '#a3f9')).toBe(true)
  })

  it('does not match a digit run from the middle of the id', () => {
    expect(matchesOrder(o, '45de')).toBe(false)
  })

  it('matches a player ID and a checkout field value', () => {
    const p = order({}, [
      item({
        playerId: '5123456789',
        fields: [
          { key: 'id', label: { en: 'Player ID' }, value: '5123456789' },
          { key: 'zone', label: { en: 'Zone' }, value: 'Zone-2211' },
        ],
      }),
    ])
    expect(matchesOrder(p, '512345')).toBe(true)
    expect(matchesOrder(p, 'zone-2211')).toBe(true)
  })

  it('matches the transfer recipient name and detail', () => {
    const t = order({}, [
      item({ fulfillmentType: 'transfer', recipient: { name: 'Ali Hassan', country: 'LB', detail: '+961 70 123 456' } }),
    ])
    expect(matchesOrder(t, 'hassan')).toBe(true)
    expect(matchesOrder(t, '70 123')).toBe(true)
  })

  it('matches the product title in English and Arabic', () => {
    expect(matchesOrder(o, 'pubg')).toBe(true)
    expect(matchesOrder(o, 'ببجي')).toBe(true)
  })

  it('returns false when nothing matches', () => {
    expect(matchesOrder(o, 'free fire')).toBe(false)
  })

  it('never matches inside a photo (QR / barcode) URL', () => {
    const photo = order({}, [
      item({
        fields: [
          {
            key: 'qr',
            label: { en: 'QR code' },
            value: 'https://salehcardstorage.blob.core.windows.net/product-images/order-files/9f8e7d.jpg',
          },
        ],
      }),
    ])
    expect(matchesOrder(photo, 'blob')).toBe(false)
    expect(matchesOrder(photo, '9f8e7d')).toBe(false)
    expect(matchesOrder(photo, 'https')).toBe(false)
  })
})
