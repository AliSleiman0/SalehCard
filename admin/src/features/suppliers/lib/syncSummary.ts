import type { TFunction } from 'i18next'
import type { SyncReport } from '../api/suppliers'

/** One line summing up a supplier check: "12 products checked · 1 hidden ·
 *  2 cost changes" (+ "warnings only, nothing changed" / supplier problems),
 *  in the admin's language. Plain words: the admin is non-technical. */
export function syncSummary(r: SyncReport, t: TFunction): string {
  const parts = [t('sup_sum_checked', { count: r.checked })]
  const w = r.dryRun // warning-only: nothing happened, say what would
  const add = (n: number, key: string, would = false) => {
    if (n > 0) parts.push(t(would && w ? `${key}_would` : key, { count: n }))
  }
  add(r.costChanges, 'sup_sum_cost')
  add(r.paused, 'sup_sum_paused', true)
  add(r.recovered, 'sup_sum_recovered', true)
  add(r.promoted, 'sup_sum_promoted', true)
  add(r.hidden, 'sup_sum_hidden', true)
  add(r.shown, 'sup_sum_shown', true)
  add(r.conflicts, 'sup_sum_conflicts')
  add(r.errors, 'sup_sum_errors')
  const down = r.suppliers.filter((s) => s.status === 'unavailable' || s.status === 'stale')
  if (down.length > 0) parts.push(t('sup_sum_down', { names: down.map((s) => s.name).join(', ') }))
  if (parts.length === 1 && r.checked > 0) parts.push(t('sup_sum_none'))
  return (r.dryRun ? t('sup_sum_warn_prefix') : '') + parts.join(' · ')
}
