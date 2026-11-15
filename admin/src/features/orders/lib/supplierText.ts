import type { TFunction } from 'i18next'
import { ApiError } from '@/lib/api-client'
import { apiErrorText } from '@/lib/apiErrorText'

/** Fixed sentences the server writes about sending an order to a supplier
 *  (why a supplier can't take it, a bulk re-send outcome), keyed to their
 *  translation. */
const FIXED: Record<string, string> = {
  "this supplier isn't connected": 'ord_msg_not_connected',
  'no supplier product linked — link one in the product editor': 'ord_msg_no_product_linked',
  "couldn't load this supplier's product list — try again in a minute": 'ord_msg_list_unavailable',
  "not in this supplier's product list any more": 'ord_msg_not_in_list',
  'unavailable at this supplier right now': 'ord_msg_unavailable_now',
  "switched off in the product's supplier list": 'ord_msg_switched_off',
  'invalid id': 'ord_msg_invalid_id',
  'listed twice': 'ord_msg_listed_twice',
  'the supplier refused it — try another supplier or refund': 'ord_msg_refused',
  'no linked supplier can take it — send it from the order page': 'ord_msg_no_link',
  'not found': 'ord_msg_not_found',
  'internal error': 'ord_msg_internal',
  // Reroute refusals with a fixed sentence (same text as their error code).
  'only orders that are still processing can be sent to a supplier': 'err_REROUTE_NOT_PROCESSING',
  'only supplier orders and single-item account-credit orders can be sent to a supplier': 'err_REROUTE_NOT_SUPPLIER_ORDER',
  "the previous supplier's outcome is unknown and it may still deliver — confirm the double-delivery risk to continue":
    'err_REROUTE_AMBIGUOUS',
  'the current supplier may still deliver this order — confirm the double-delivery risk to send it again':
    'err_REROUTE_NEEDS_ACK',
  "this order's quantity can't be sent to a supplier — complete or refund it by hand": 'err_REROUTE_BAD_QUANTITY',
  'choose the supplier product to order': 'err_REROUTE_UPSTREAM_REQUIRED',
  'that supplier link has no supplier product — link one in the product editor': 'ord_msg_link_no_product',
  'that supplier is not configured': 'err_REROUTE_UNKNOWN_SUPPLIER',
  'that supplier is no longer linked to the product — reload the order': 'err_REROUTE_LINK_UNKNOWN',
  "that supplier is switched off in the product's supplier list — switch it on there first": 'err_REROUTE_LINK_DISABLED',
  'the order changed while you were rerouting it — reload and try again': 'err_REROUTE_CONFLICT',
  "couldn't load that supplier's product list to check the product — try again in a minute":
    'err_REROUTE_CATALOG_UNAVAILABLE',
  'that supplier marks this product unavailable right now — choose another supplier': 'err_REROUTE_UPSTREAM_UNAVAILABLE',
  'no orders selected': 'err_REROUTE_BULK_EMPTY',
}

const NO_LINK_PREFIX = 'no linked supplier can take it: '

type Pattern = [RegExp, string, (m: RegExpMatchArray) => Record<string, string>]

const PATTERNS: Pattern[] = [
  [/^doesn't accept quantity ([\d,]+)$/, 'ord_msg_qty_not_accepted', (m) => ({ n: m[1] })],
  [
    /^product "(.*)" is not in that supplier's catalog — pick the product from the list$/,
    'ord_msg_upstream_unknown',
    (m) => ({ id: m[1] }),
  ],
  [
    /^this order sends quantity (\d+), which that supplier product doesn't accept$/,
    'ord_msg_qty_rejected',
    (m) => ({ n: m[1] }),
  ],
  [
    /^that supplier's balance \((.+?)\) is below this order's cost \((.+?)\) — top it up, or confirm to send anyway$/,
    'ord_msg_low_balance',
    (m) => ({ balance: m[1], cost: m[2] }),
  ],
  [/^re-send at most (\d+) orders at once$/, 'ord_msg_bulk_limit', (m) => ({ n: m[1] })],
]

/** A server sentence about sending an order to a supplier, in the admin's
 *  language. English and unrecognised text (e.g. a supplier's own words) come
 *  back unchanged; `code` (the error code) is the last resort when it has a
 *  translation. */
export function supplierText(msg: string, t: TFunction, lang: string, code?: string): string {
  if (!msg || lang.startsWith('en')) return msg
  const fixed = FIXED[msg]
  if (fixed) return t(fixed)
  if (msg.startsWith(NO_LINK_PREFIX)) {
    return t('ord_msg_no_link_because', { why: supplierText(msg.slice(NO_LINK_PREFIX.length), t, lang) })
  }
  for (const [re, key, values] of PATTERNS) {
    const m = msg.match(re)
    if (m) return t(key, values(m))
  }
  if (code && t(`err_${code}`, { defaultValue: '' })) return t(`err_${code}`)
  return msg
}

/** apiErrorText for a reroute request: a refusal that carries values (a low
 *  balance, a rejected quantity) keeps them in the translation. */
export function supplierErrorText(e: unknown, fallbackKey: string, t: TFunction, lang: string): string {
  if (e instanceof ApiError && e.message) return supplierText(e.message, t, lang, e.code)
  return apiErrorText(e, fallbackKey)
}
