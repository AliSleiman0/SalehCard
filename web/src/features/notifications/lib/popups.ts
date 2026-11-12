import type { AppNotification } from '../types'

/** Kinds that pop a banner at the top of the screen (client feedback #64). */
export const POPUP_KINDS = new Set(['topup_approved', 'topup_rejected'])

/** Unread top-up notifications created after `sinceMs` (epoch ms), oldest
 * first — the ones to pop. `sinceMs` is the page-load time, then advanced to
 * the newest popped note, so a note never pops twice and old ones never pop. */
export function newPopups(list: AppNotification[], sinceMs: number): AppNotification[] {
  return list
    .filter((n) => POPUP_KINDS.has(n.kind) && !n.readAt && Date.parse(n.createdAt) > sinceMs)
    .sort((a, b) => Date.parse(a.createdAt) - Date.parse(b.createdAt))
}
