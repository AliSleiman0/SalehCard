import { create } from 'zustand'

export interface CartItem {
  key: string
  id: string
  brand: string
  title: string
  art: string
  variant: string
  price: number
  qty: number
  pid?: string
  fulfill: 'code' | 'credit' | 'transfer'
  recipient?: { name: string; country: string; detail: string } | null
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
    set((state) => {
      const key = `${item.id}|${item.variant}`
      const existing = state.items.find((x) => x.key === key)
      if (existing) {
        return {
          items: state.items.map((x) =>
            x.key === key ? { ...x, qty: x.qty + item.qty } : x,
          ),
        }
      }
      return { items: [...state.items, { ...item, key }] }
    })
  },
  remove(key: string) {
    set((state) => ({ items: state.items.filter((x) => x.key !== key) }))
  },
  setQty(key: string, qty: number) {
    set((state) => ({
      items: state.items.map((x) => (x.key === key ? { ...x, qty } : x)),
    }))
  },
  clear() {
    set({ items: [] })
  },
}))

export const useCartCount = () =>
  useCartStore((s) => s.items.reduce((n, x) => n + x.qty, 0))
