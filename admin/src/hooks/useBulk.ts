import { useState } from 'react'

/** Bulk selection over a list of ids (the prototype's `useBulk`). */
export function useBulk(ids: string[]) {
  const [sel, setSel] = useState<string[]>([])
  const toggle = (id: string) => setSel((s) => (s.includes(id) ? s.filter((x) => x !== id) : [...s, id]))
  const all = ids.length > 0 && sel.length === ids.length
  const toggleAll = () => setSel(all ? [] : [...ids])
  const clear = () => setSel([])
  return { sel, toggle, all, toggleAll, clear, some: sel.length > 0 }
}
