import { describe, it, expect } from 'vitest'
import { buildRecipient, buildOrderLine, isValidLebaneseMobile, nonQuantityFields } from './orderFields'
import type { CartItem } from '@/stores/cart'
import type { InputField, Product } from '@/types'

function field(key: string, type: InputField['type'] = 'text'): InputField {
  return { key, label: { en: key }, type, sensitive: false }
}

function product(over: Partial<Product>): Product {
  return {
    id: 'p1',
    title: { en: 'P', ar: 'P', tr: 'P' },
    category: 'games',
    images: [],
    thumbnail: '',
    variants: [],
    fulfillmentType: 'account_credit',
    stock: 5,
    available: true,
    ratings: { average: 0, count: 0 },
    createdAt: '',
    updatedAt: '',
    ...over,
  }
}

const cartItem: CartItem = {
  key: 'p1|v1',
  id: 'p1',
  variantId: 'v1',
  brand: 'Brand',
  title: 'Title',
  art: 'soft',
  variant: '100',
  price: 10,
  qty: 2,
  fulfill: 'credit',
}

describe('nonQuantityFields', () => {
  it('drops legacy quantity fields', () => {
    const p = product({ inputFields: [field('id'), field('qty', 'quantity'), field('zone')] })
    expect(nonQuantityFields(p).map((f) => f.key)).toEqual(['id', 'zone'])
  })
  it('returns [] for a product with no fields', () => {
    expect(nonQuantityFields(product({}))).toEqual([])
    expect(nonQuantityFields(undefined)).toEqual([])
  })
})

describe('isValidLebaneseMobile', () => {
  it('accepts valid prefixes', () => {
    expect(isValidLebaneseMobile('70123456')).toBe(true)
    expect(isValidLebaneseMobile('03 123 456')).toBe(true) // 7-digit 3-prefix re-expands to 03
    expect(isValidLebaneseMobile('+96181123456')).toBe(true)
    expect(isValidLebaneseMobile('0096171123456')).toBe(true)
  })
  it('rejects bad numbers', () => {
    expect(isValidLebaneseMobile('12345678')).toBe(false)
    expect(isValidLebaneseMobile('7012345')).toBe(false)
    expect(isValidLebaneseMobile('')).toBe(false)
  })
})

describe('buildRecipient', () => {
  it('maps by key-substring heuristic', () => {
    const r = buildRecipient({ recipientName: 'Ayse', country: 'Türkiye', accountNumber: 'TR1234' })
    expect(r).toEqual({ name: 'Ayse', country: 'Türkiye', detail: 'TR1234' })
  })
  it('falls back to joined values when name/detail keys are absent', () => {
    const r = buildRecipient({ foo: 'a', bar: 'b' })
    expect(r.name).toBe('a / b')
    expect(r.detail).toBe('a / b')
    expect(r.country).toBe('')
  })
})

describe('buildOrderLine', () => {
  it('account_credit → fields[] + playerId from the first filled field', () => {
    const p = product({ fulfillmentType: 'account_credit', inputFields: [field('playerId'), field('zone')] })
    const line = buildOrderLine(cartItem, p, { playerId: '900123', zone: '5' })
    expect(line.fields).toEqual([
      { key: 'playerId', value: '900123' },
      { key: 'zone', value: '5' },
    ])
    expect(line.playerId).toBe('900123')
    expect(line.qty).toBe(2)
  })
  it('account_credit → omits empty fields', () => {
    const p = product({ fulfillmentType: 'account_credit', inputFields: [field('playerId'), field('zone')] })
    const line = buildOrderLine(cartItem, p, { playerId: '900123', zone: '  ' })
    expect(line.fields).toEqual([{ key: 'playerId', value: '900123' }])
  })
  it('transfer → recipient from the heuristic', () => {
    const p = product({ fulfillmentType: 'transfer', inputFields: [field('name'), field('country', 'select'), field('iban')] })
    const line = buildOrderLine(cartItem, p, { name: 'Sam', country: 'Egypt', iban: 'EG99' })
    expect(line.recipient).toEqual({ name: 'Sam', country: 'Egypt', detail: 'EG99' })
    expect(line.fields).toBeUndefined()
  })
  it('code / no fields → carries the pre-collected pid', () => {
    const p = product({ fulfillmentType: 'code', inputFields: [] })
    const line = buildOrderLine({ ...cartItem, pid: 'CODE1' }, p, {})
    expect(line.playerId).toBe('CODE1')
    expect(line.fields).toBeUndefined()
  })
})
