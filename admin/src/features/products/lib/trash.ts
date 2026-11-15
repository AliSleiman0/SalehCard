import i18n from '@/i18n/config'
import type { BulkAction } from '../api/products'

/**
 * Soft delete ("trash") rules for the admin product list, kept pure so they are
 * unit-testable without rendering the page.
 *
 * - The normal list never shows trashed products; the Deleted view shows only them.
 * - "Delete" moves to the trash (reversible). Restore returns each product to
 *   its previous active/draft state.
 * - Permanent delete exists only in the Deleted view, only for super admins,
 *   and only for explicitly selected rows (never "all matching").
 */

/** Bulk-bar actions offered for the current view. */
export function bulkActionsFor(trashView: boolean, isSuperAdmin: boolean, matchAll: boolean): BulkAction[] {
  if (!trashView) return ['activate', 'deactivate', 'assign-category', 'delete']
  return ['restore', 'assign-category', ...(isSuperAdmin && !matchAll ? (['purge'] as BulkAction[]) : [])]
}

/** Offer "Select all N matching" once every row on the page is selected and
 *  more products match than the page shows. */
export function canSelectAllMatching(allOnPageSelected: boolean, shown: number, total: number): boolean {
  return allOnPageSelected && total > shown
}

/** i18n options for a product count: `count` picks the English plural, `n` is
 *  the grouped number shown ("1,240"). */
const counted = (n: number) => ({ count: n, n: n.toLocaleString() })

/**
 * Confirmation text for a bulk action, or null when it needs none. Trash and
 * permanent delete always confirm; anything applied to "all matching" confirms
 * with the full count, since it reaches beyond the rows on screen.
 */
export function bulkConfirmText(action: BulkAction, count: number, matchAll: boolean): string | null {
  const all = matchAll ? '_all' : ''
  switch (action) {
    case 'delete':
    case 'purge':
      return i18n.t(`prod_confirm_${action}${all}`, counted(count))
    case 'restore':
    case 'activate':
    case 'deactivate':
      return matchAll ? i18n.t(`prod_confirm_${action}_all`, counted(count)) : null
    default:
      return null
  }
}

/** Toast after a successful bulk action. */
export function bulkDoneText(action: BulkAction, modified: number): string {
  switch (action) {
    case 'delete':
    case 'restore':
    case 'purge':
    case 'activate':
    case 'deactivate':
      return i18n.t(`prod_done_${action}`, counted(modified))
    default:
      return i18n.t('prod_done_updated', counted(modified))
  }
}
