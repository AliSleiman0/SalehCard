import { ffKey, artForCategory, type FfKey } from '@/components'
import { relativeTime } from '@/lib/utils'
import type { AdminOrder, AdminOrderItem, OrderCustomer } from '../api/orders'

/** The flat shape the order list/detail tables render. */
export interface OrderView {
  id: string
  customer: string
  email: string
  product: string
  qty: number
  art: string
  ff: FfKey
  amount: number
  cur: 'USD' | 'TRY'
  pay: string
  status: string
  date: string
  raw: AdminOrder
}

/** Display name for a customer — the User model has no name field, so we derive
 *  one from the email local-part, falling back to phone. */
export function customerName(c: OrderCustomer | null): string {
  if (c?.email) return c.email.split('@')[0]
  if (c?.phone) return c.phone
  return 'Unknown'
}

const unitsFmt = new Intl.NumberFormat('en-US', { maximumFractionDigits: 2 })

/** The quantity caption for one order line: a per-unit line shows the amount
 *  the customer typed ("45,000 units", qty is always 1 there); a package line
 *  shows "Qty N". */
export function itemQtyLabel(it: Pick<AdminOrderItem, 'qty' | 'amount'>): string {
  return it.amount != null && it.amount > 0 ? `${unitsFmt.format(it.amount)} units` : `Qty ${it.qty}`
}

/** One account-credit line the operator has to credit by hand. */
export interface CreditLine {
  title: string
  /** The account to credit: the line's playerId, else its first captured field. */
  playerId: string
  qtyLabel: string
  /** Whole line price (price × qty). */
  total: number
}

/** The account-credit lines of an order plus how many distinct accounts they
 *  target. An order can hold several lines for different player IDs (the cart
 *  keeps one line per ID), and completion is order-level — so the operator
 *  must see and credit every one of them, not just the first. */
export function creditLines(items: AdminOrderItem[]): { lines: CreditLine[]; distinctIds: number } {
  const lines = items
    .filter((it) => it.fulfillmentType === 'account_credit')
    .map((it) => ({
      title: it.title.en,
      playerId: (it.playerId || it.fields?.find((f) => f.value)?.value || '').trim(),
      qtyLabel: itemQtyLabel(it),
      total: it.price * it.qty,
    }))
  const distinctIds = new Set(lines.map((l) => l.playerId).filter(Boolean)).size
  return { lines, distinctIds }
}

/** One line's delivered codes (one code per unit). */
export interface CodeLine {
  title: string
  codes: string[]
}

/** The delivered codes of an order, grouped per line. A mixed order (code
 *  lines + manual credit/transfer lines) carries its codes while still
 *  processing. Old orders have no items[].codes — they fall back to the single
 *  fulfillment.deliveredCode, titled after the first code line. */
export function codeLines(items: AdminOrderItem[], deliveredCode?: string): CodeLine[] {
  const lines = items
    .filter((it) => it.codes && it.codes.length > 0)
    .map((it) => ({ title: it.title.en, codes: it.codes! }))
  if (lines.length > 0 || !deliveredCode) return lines
  const codeItem = items.find((it) => it.fulfillmentType === 'code') ?? items[0]
  return [{ title: codeItem?.title.en ?? '', codes: [deliveredCode] }]
}

/** PayChip keys are wallet/visa/usdt; the API stores card → map it to visa. */
function payKey(method: string): string {
  return method === 'card' ? 'visa' : method
}

/** Map an enriched API order to the flat view the admin pages render. */
export function adaptOrder(o: AdminOrder): OrderView {
  const first = o.items[0]
  const extra = o.items.length - 1
  const product = first
    ? first.title.en + (extra > 0 ? ` +${extra} more` : '')
    : '—'
  const qty = o.items.reduce((sum, it) => sum + it.qty, 0)

  return {
    id: o.id,
    customer: customerName(o.customer),
    email: o.customer?.email ?? '',
    product,
    qty,
    art: first ? artForCategory(first.category) : 'soft',
    ff: first ? ffKey(first.fulfillmentType) : 'code',
    amount: o.total,
    cur: (o.currency as 'USD' | 'TRY') || 'USD',
    pay: payKey(o.paymentMethod),
    status: o.status,
    date: relativeTime(o.createdAt),
    raw: o,
  }
}
