import type { SyncReport } from '../api/suppliers'

/** One line summing up a supplier sync pass: "12 products checked · 1 hidden ·
 *  2 cost changes" (+ "dry run — nothing changed" / supplier problems). */
export function syncSummary(r: SyncReport): string {
  const parts = [`${r.checked} product${r.checked === 1 ? '' : 's'} checked`]
  const add = (n: number, one: string, many = one) => {
    if (n > 0) parts.push(`${n} ${n === 1 ? one : many}`)
  }
  add(r.costChanges, 'cost change', 'cost changes')
  add(r.paused, 'link paused (loss)', 'links paused (loss)')
  add(r.recovered, 'link profitable again', 'links profitable again')
  add(r.promoted, 'supplier switched', 'suppliers switched')
  add(r.hidden, 'hidden')
  add(r.shown, 'back on sale')
  add(r.conflicts, 'skipped (edited meanwhile)')
  add(r.errors, 'error', 'errors')
  const down = r.suppliers.filter((s) => s.status === 'unavailable' || s.status === 'stale')
  if (down.length > 0) parts.push(`couldn't reach ${down.map((s) => s.name).join(', ')}`)
  if (parts.length === 1 && r.checked > 0) parts.push('no changes')
  return (r.dryRun ? 'dry run, nothing changed — ' : '') + parts.join(' · ')
}
