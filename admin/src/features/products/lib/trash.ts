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

const plural = (n: number) => `${n.toLocaleString()} product${n === 1 ? '' : 's'}`

/**
 * Confirmation text for a bulk action, or null when it needs none. Trash and
 * permanent delete always confirm; anything applied to "all matching" confirms
 * with the full count, since it reaches beyond the rows on screen.
 */
export function bulkConfirmText(action: BulkAction, count: number, matchAll: boolean): string | null {
  const what = matchAll ? `all ${plural(count)} matching the current filters` : plural(count)
  switch (action) {
    case 'delete':
      return `Move ${what} to the trash? Customers won't see or be able to buy them until you restore them from the Deleted view.`
    case 'purge':
      return `Permanently delete ${what}? This erases them for good and cannot be undone. Past orders keep their own copy of the product.`
    case 'restore':
      return matchAll ? `Restore ${what}? Each goes back to its previous status (active or draft).` : null
    case 'activate':
      return matchAll ? `Activate ${what}? Customers will see them.` : null
    case 'deactivate':
      return matchAll ? `Deactivate ${what}? Customers won't see them.` : null
    default:
      return null
  }
}

/** Toast after a successful bulk action. */
export function bulkDoneText(action: BulkAction, modified: number): string {
  switch (action) {
    case 'delete':
      return `Moved ${plural(modified)} to the trash.`
    case 'restore':
      return `Restored ${plural(modified)}.`
    case 'purge':
      return `Permanently deleted ${plural(modified)}.`
    case 'activate':
      return `Activated ${plural(modified)}.`
    case 'deactivate':
      return `Deactivated ${plural(modified)}.`
    default:
      return `Updated ${plural(modified)}.`
  }
}
