import type { Order } from '@/types'
import { orderNumber } from '@/lib/orderNumber'

// Photo field values (QR / barcode uploads) and any other link are never
// searched: matching "https" or a blob host would light up unrelated orders.
const URL_RE = /^https?:\/\//i

const fold = (s: string) => s.trim().toLocaleLowerCase()

// normalizeOrderQuery trims, lower-cases and drops leading "#"s, so "#A3F9C1",
// " a3f9c1 " and "a3f9c1" all search the same thing.
export function normalizeOrderQuery(raw: string): string {
  return fold(raw).replace(/^#+/, '').trim()
}

// matchesOrder is the customer order-history search rule (client #79), shared
// in spirit with the app's `orderMatchesQuery`:
//   - the order number shown on every row and order page ("#8D03458D", see
//     orderNumber) or a prefix of it, and the full order id or a prefix of it
//     (the number is the id's last 8 characters, so a suffix of the id too);
//   - per line: player ID, checkout field values, transfer recipient name +
//     detail, and the product title in every locale.
// Case-insensitive; an empty query matches everything. Values are matched as
// substrings, the id only at its ends (a bare "12" would otherwise hit most
// hex ids).
export function matchesOrder(o: Order, rawQuery: string): boolean {
  const q = normalizeOrderQuery(rawQuery)
  if (!q) return true

  const id = o.id.toLowerCase()
  const number = orderNumber(o.id).slice(1).toLowerCase()
  if (number.startsWith(q) || id.startsWith(q) || id.endsWith(q)) {
    return true
  }

  const hit = (v?: string) => !!v && !URL_RE.test(v.trim()) && fold(v).includes(q)

  if (hit(o.fulfillment?.transferRef)) return true
  for (const it of o.items) {
    if (hit(it.playerId) || hit(it.recipient?.name) || hit(it.recipient?.detail)) return true
    if (it.fields?.some((f) => hit(f.value))) return true
    if (Object.values(it.title ?? {}).some((t) => hit(t))) return true
  }
  return false
}
