import { describe, it, expect } from 'vitest'
import { adaptProduct } from './adaptProduct'
import type { InputField, Product } from '@/types'

function product(over: Partial<Product>): Product {
  return {
    id: 'p1',
    title: { en: 'Up Live', ar: '', tr: '' },
    category: 'credits',
    images: [],
    thumbnail: '',
    variants: [{ id: 'v1', denomination: '1,000', price: 1.5 }],
    fulfillmentType: 'account_credit',
    stock: 0,
    available: true,
    ratings: { average: 0, count: 0 },
    createdAt: '',
    updatedAt: '',
    ...over,
  } as Product
}

const amountField: InputField = {
  key: 'amount',
  label: { en: 'Enter Amount' },
  type: 'amount',
  sensitive: false,
  constraints: { min: 100, max: 200000 },
}
const idField: InputField = { key: 'id', label: { en: 'ID' }, type: 'text', sensitive: false }
const qtyField: InputField = { key: 'qty', label: { en: 'Qty' }, type: 'quantity', sensitive: false }

describe('adaptProduct needsId', () => {
  it('keeps the ID box for a unit-priced product whose only field is the amount', () => {
    const v = adaptProduct(
      product({ inputFields: [amountField], unitPricing: { fieldKey: 'amount', blockUnits: 1000 } }),
      'en',
    )
    expect(v.unit).toBeDefined()
    expect(v.inputFields).toEqual([])
    expect(v.needsId).toBe(true)
    expect(v.idLabel).toBe('Account / Player ID')
  })
  it('keeps the ID box for a legacy product whose only field is a quantity row', () => {
    const v = adaptProduct(product({ inputFields: [qtyField] }), 'en')
    expect(v.inputFields).toEqual([])
    expect(v.needsId).toBe(true)
  })
  it('uses the product schema instead when it has a customer-typed field', () => {
    const v = adaptProduct(
      product({
        inputFields: [idField, amountField],
        unitPricing: { fieldKey: 'amount', blockUnits: 1000 },
      }),
      'en',
    )
    expect(v.inputFields).toEqual([idField])
    expect(v.needsId).toBe(false)
    expect(v.idLabel).toBeUndefined()
  })
  it('never asks for an ID on code products', () => {
    const v = adaptProduct(product({ fulfillmentType: 'code', inputFields: [] }), 'en')
    expect(v.needsId).toBe(false)
  })
})

describe('adaptProduct category label', () => {
  it('never shows a raw legacy slug like "Cat 447"', () => {
    expect(adaptProduct(product({ category: 'cat-447' }), 'en').title).toBe('')
  })

  it("prefers the assigned node's localized name", () => {
    const p = product({
      category: 'cat-447',
      categoryName: { en: 'Dahab Transfer', ar: 'تحويل ذهب', tr: '' },
    })
    expect(adaptProduct(p, 'en').title).toBe('Dahab Transfer')
    expect(adaptProduct(p, 'ar').title).toBe('تحويل ذهب')
    expect(adaptProduct(p, 'tr').title).toBe('Dahab Transfer') // falls back to en
  })

  it('still humanizes ordinary flat categories', () => {
    expect(adaptProduct(product({ category: 'gift-cards' }), 'en').title).toBe('Gift Cards')
  })
})
