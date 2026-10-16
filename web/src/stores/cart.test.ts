import { describe, it, expect } from 'vitest'
import { addItem, cartKey, type CartItem } from './cart'

const base: Omit<CartItem, 'key'> = {
  id: 'p1',
  variantId: 'v1',
  brand: 'Brand',
  title: 'Title',
  art: 'soft',
  variant: '100',
  price: 10,
  qty: 1,
  fulfill: 'credit',
}

describe('cartKey', () => {
  it('includes the amount and the player id', () => {
    expect(cartKey(base)).toBe('p1|v1||')
    expect(cartKey({ ...base, amount: 45000 })).toBe('p1|v1|45000|')
    expect(cartKey({ ...base, amount: 45000, pid: ' 123 ' })).toBe('p1|v1|45000|123')
  })
})

describe('addItem', () => {
  it('merges quantity lines, clamped to the max', () => {
    let items = addItem([], { ...base, qty: 2, qtyMax: 3 })
    items = addItem(items, { ...base, qty: 2, qtyMax: 3 })
    expect(items).toHaveLength(1)
    expect(items[0].qty).toBe(3)
  })
  it('keeps different amounts as separate lines', () => {
    let items = addItem([], { ...base, amount: 45000, price: 67.5 })
    items = addItem(items, { ...base, amount: 1000, price: 1.5 })
    expect(items.map((x) => x.key)).toEqual(['p1|v1|45000|', 'p1|v1|1000|'])
  })
  it('pins amount lines to qty 1, even when re-added', () => {
    let items = addItem([], { ...base, amount: 45000, price: 67.5, qty: 3 })
    expect(items[0].qty).toBe(1)
    items = addItem(items, { ...base, amount: 45000, price: 70, qty: 2 })
    expect(items).toHaveLength(1)
    expect(items[0].qty).toBe(1)
    expect(items[0].price).toBe(70)
  })
  it('keeps the same amount for different player IDs as separate lines', () => {
    let items = addItem([], { ...base, amount: 45000, price: 67.5, pid: 'A1' })
    items = addItem(items, { ...base, amount: 45000, price: 67.5, pid: 'B2' })
    expect(items).toHaveLength(2)
    expect(items.map((x) => x.pid)).toEqual(['A1', 'B2'])
  })
  it('never merges quantity lines for different player IDs', () => {
    let items = addItem([], { ...base, qty: 2, pid: 'A1' })
    items = addItem(items, { ...base, qty: 3, pid: 'B2' })
    items = addItem(items, { ...base, qty: 1, pid: 'A1' })
    expect(items.map((x) => [x.pid, x.qty])).toEqual([
      ['A1', 3],
      ['B2', 3],
    ])
  })
})
