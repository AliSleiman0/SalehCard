import type { TFunction } from 'i18next'

/** The fixed sentences the server writes into an order's timeline notes (and
 *  a bridge command's fail reason), keyed to their translation. */
const FIXED: Record<string, string> = {
  'charge reversal failed': 'tl_note_reversal_failed',
  'mobile recharge delivered by bridge device': 'tl_note_bridge_delivered',
  'USDT payment confirmed on-chain': 'tl_note_usdt_confirmed',
  'awaiting the new supplier': 'tl_note_awaiting_new_supplier',
  'awaiting upstream provider': 'tl_note_awaiting_supplier',
  'awaiting provider integration': 'tl_note_awaiting_integration',
  'upstream temporarily unavailable': 'tl_note_supplier_unavailable',
  'queued for bridge device': 'tl_note_queued_bridge',
  'queued for manual recharge': 'tl_note_queued_manual',
  'dispatched to bridge device': 'tl_note_dispatched_bridge',
  'stuck upstream — needs attention': 'tl_note_stuck_plain',
  'supplier delivered after the order was rerouted or left processing — reconcile manually': 'tl_note_late_delivery',
  'completed by the new supplier': 'tl_note_completed_new_supplier',
  'completed by supplier check': 'tl_note_completed_check',
  'completed by supplier re-dispatch': 'tl_note_completed_redispatch',
  'rejected by the backup supplier — refunded': 'tl_note_rejected_backup_refunded',
  'rejected upstream — refunded': 'tl_note_rejected_refunded',
  'rejected upstream on re-dispatch — refunded': 'tl_note_rejected_redispatch_refunded',
  underpaid: 'tl_note_underpaid',
  'payment expired': 'tl_note_payment_expired',
  'mobile recharge failed on the device': 'tl_note_bridge_failed',
  'recharge attempts exhausted': 'tl_note_recharge_exhausted',
  'recharge timed out waiting for a device': 'tl_note_recharge_timeout',
  'recharge only partially completed': 'tl_note_recharge_partial',
  // The reasons after "stuck upstream — needs attention: ".
  'rerouted order has no sendable supplier line': 'tl_why_rerouted_no_line',
  'rejected by the new supplier': 'tl_why_rejected_new',
  'upstream status unavailable past give-up window': 'tl_why_status_unavailable',
  'rejected by the supplier': 'tl_why_rejected',
  'still waiting upstream past give-up window': 'tl_why_still_waiting',
  're-dispatch retries exhausted': 'tl_why_retries_exhausted',
  'retryable order has no sendable supplier line': 'tl_why_no_line',
}

const STUCK_PREFIX = 'stuck upstream — needs attention: '

type Pattern = [RegExp, string, (m: RegExpMatchArray, t: TFunction) => Record<string, string>]

const PATTERNS: Pattern[] = [
  [/^codes delivered for (\d+) line\(s\)$/, 'tl_note_codes_delivered', (m) => ({ n: m[1] })],
  [
    /^supplier (#\d+|\(none\)) couldn't take the order — trying supplier #(\d+) automatically$/,
    'tl_note_fallback',
    (m, t) => ({ from: m[1] === '(none)' ? t('tl_none') : m[1], to: m[2] }),
  ],
  [/^sent to another supplier \(#(\d+)\) by admin$/, 'tl_note_sent_another', (m) => ({ n: m[1] })],
  [/^sent to supplier \(#(\d+)\) by admin$/, 'tl_note_sent_supplier', (m) => ({ n: m[1] })],
  [/^sent again to supplier \(#(\d+)\) by admin$/, 'tl_note_sent_again', (m) => ({ n: m[1] })],
  [/^device reported failure code (-?\d+)$/, 'tl_note_device_code', (m) => ({ n: m[1] })],
]

/** A timeline note (or bridge fail reason) in the admin's language. The server
 *  writes English; its fixed sentences are recognised and translated. English,
 *  notes an admin typed and anything unrecognised come back unchanged. In a
 *  "stuck" note only the prefix is translated when the reason after it is the
 *  supplier's own text. */
export function timelineNote(note: string, t: TFunction, lang: string): string {
  if (!note || lang.startsWith('en')) return note
  const fixed = FIXED[note]
  if (fixed) return t(fixed)
  if (note.startsWith(STUCK_PREFIX)) {
    const why = note.slice(STUCK_PREFIX.length)
    return t('tl_note_stuck', { why: FIXED[why] ? t(FIXED[why]) : why })
  }
  for (const [re, key, values] of PATTERNS) {
    const m = note.match(re)
    if (m) return t(key, values(m, t))
  }
  return note
}

/** A timeline status label ("created", "stuck_upstream", …) in the admin's language. */
export function timelineStatus(status: string, t: TFunction): string {
  return t(`tl_status_${status}`, { defaultValue: status })
}
