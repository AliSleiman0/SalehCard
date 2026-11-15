import type { SyncReport } from '../api/suppliers'

/** One line summing up a supplier check: "12 products checked · 1 hidden ·
 *  2 cost changes" (+ "warnings only, nothing changed" / supplier problems).
 *  Plain words: the admin is non-technical. */
export function syncSummary(r: SyncReport): string {
  const parts = [`${r.checked} product${r.checked === 1 ? '' : 's'} checked`]
  const add = (n: number, one: string, many = one) => {
    if (n > 0) parts.push(`${n} ${n === 1 ? one : many}`)
  }
  const w = r.dryRun // warning-only: nothing happened, say what would
  add(r.costChanges, 'cost change', 'cost changes')
  add(
    r.paused,
    w ? 'supplier would be paused (loss)' : 'supplier paused (loss)',
    w ? 'suppliers would be paused (loss)' : 'suppliers paused (loss)'
  )
  add(
    r.recovered,
    w ? 'supplier would be unpaused' : 'supplier profitable again',
    w ? 'suppliers would be unpaused' : 'suppliers profitable again'
  )
  add(
    r.promoted,
    w ? 'product would switch supplier' : 'product switched supplier',
    w ? 'products would switch supplier' : 'products switched supplier'
  )
  add(r.hidden, w ? 'would go off sale' : 'taken off sale')
  add(r.shown, w ? 'would go back on sale' : 'back on sale')
  add(r.conflicts, 'skipped (edited meanwhile)')
  add(r.errors, 'error', 'errors')
  const down = r.suppliers.filter((s) => s.status === 'unavailable' || s.status === 'stale')
  if (down.length > 0) parts.push(`couldn't reach ${down.map((s) => s.name).join(', ')}`)
  if (parts.length === 1 && r.checked > 0) parts.push('no changes')
  return (r.dryRun ? 'warnings only, nothing changed — ' : '') + parts.join(' · ')
}
