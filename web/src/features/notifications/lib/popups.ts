import type { AppNotification } from '../types'

/** Kinds that pop a banner at the top of the screen (client feedback #64). */
export const POPUP_KINDS = new Set(['topup_approved', 'topup_rejected'])

/** Unread top-up notifications created after `sinceMs` (epoch ms), oldest
 * first — the ones to pop. `sinceMs` is seeded from the inbox (baselineFrom), then advanced to
 * the newest popped note, so a note never pops twice and old ones never pop. */
export function newPopups(list: AppNotification[], sinceMs: number): AppNotification[] {
  return list
    .filter((n) => POPUP_KINDS.has(n.kind) && !n.readAt && Date.parse(n.createdAt) > sinceMs)
    .sort((a, b) => Date.parse(a.createdAt) - Date.parse(b.createdAt))
}

/** The pop-up baseline: the newest notification time in the inbox (epoch ms),
 * or 0 when it is empty. Taken from server timestamps, not the device clock,
 * so a phone whose clock runs ahead still pops notes created after load. */
export function baselineFrom(list: AppNotification[]): number {
  let max = 0
  for (const n of list) {
    const t = Date.parse(n.createdAt)
    if (Number.isFinite(t) && t > max) max = t
  }
  return max
}
