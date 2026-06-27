import { ffKey, artForCategory, type FfKey } from '@/components'
import { relativeTime } from '@/lib/utils'
import type { AdminOrder, OrderCustomer } from '../api/orders'

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
