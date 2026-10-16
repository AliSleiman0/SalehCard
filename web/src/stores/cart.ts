import { create } from 'zustand'

export interface CartItem {
  key: string
  id: string
  variantId: string
  brand: string
  title: string
  art: string
  /** real product image (thumbnail); art is the fallback */
  image?: string
  variant: string
  /** unit price (for amount lines: the whole line price) */
  price: number
  qty: number
  /** purchase-quantity bounds (legacy quantity field); default 1..99 */
  qtyMin?: number
  qtyMax?: number
  /** customer-typed amount on a unit-priced line; such lines are always qty 1 */
  amount?: number
  pid?: string
  fulfill: 'code' | 'credit' | 'transfer'
  recipient?: { name: string; country: string; detail: string } | null
}

/** Cart line identity: different typed amounts — or different player IDs —
 * never merge into one line (merging would silently drop one of the IDs). */
export function cartKey(item: Pick<CartItem, 'id' | 'variantId' | 'amount' | 'pid'>): string {
  return `${item.id}|${item.variantId}|${item.amount ?? ''}|${(item.pid ?? '').trim()}`
}

/** Pure add/merge step (exported for tests). Amount lines stay qty 1 — re-adding
 * the same amount for the same ID refreshes the line instead of doubling it;
 * quantity lines for the same ID merge their qty, clamped to the line's max. */
export function addItem(items: CartItem[], item: Omit<CartItem, 'key'>): CartItem[] {
  const key = cartKey(item)
  const isAmount = item.amount !== undefined
  const next: CartItem = { ...item, key, qty: isAmount ? 1 : item.qty }
  const existing = items.find((x) => x.key === key)
  if (!existing) return [...items, next]
  return items.map((x) => {
    if (x.key !== key) return x
    if (isAmount) return next
    const max = item.qtyMax ?? x.qtyMax
    const qty = x.qty + item.qty
    return { ...x, qty: max !== undefined ? Math.min(max, qty) : qty }
  })
}

interface CartState {
  items: CartItem[]
  add(item: Omit<CartItem, 'key'>): void
  remove(key: string): void
  setQty(key: string, qty: number): void
  clear(): void
}

export const useCartStore = create<CartState>((set) => ({
  items: [],
  add(item: Omit<CartItem, 'key'>) {
    set((state) => ({ items: addItem(state.items, item) }))
  },
  remove(key: string) {
    set((state) => ({ items: state.items.filter((x) => x.key !== key) }))
  },
  setQty(key: string, qty: number) {
    set((state) => ({
      // amount lines are pinned to qty 1 (the server rejects anything else)
      items: state.items.map((x) => (x.key === key && x.amount === undefined ? { ...x, qty } : x)),
    }))
  },
  clear() {
    set({ items: [] })
  },
}))

export const useCartCount = () =>
  useCartStore((s) => s.items.reduce((n, x) => n + x.qty, 0))
