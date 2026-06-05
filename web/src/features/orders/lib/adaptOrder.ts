import { artForCategory } from '@/lib/art'
import type { Order, OrderStatus, PaymentMethod } from '@/types'
import type { OrderView } from '../types'
import { toFulfillKind } from './fulfillment'

const METHOD_LABEL: Record<PaymentMethod, string> = {
  wallet: 'Wallet',
  card: 'Visa',
  usdt: 'USDT',
}

// statusLabel maps a backend OrderStatus to the presentational status string the
// OrderView components expect (and the i18n keys that exist for them).
function statusLabel(s: OrderStatus): string {
  switch (s) {
    case 'completed':
      return 'completed'
    case 'pending':
    case 'processing':
      return 'processing'
    default:
      return s // failed | refunded
  }
}

function fmtDate(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return (
    d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' }) +
    ' · ' +
    d.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })
  )
}

// adaptOrder maps a backend Order to the presentational OrderView consumed by
// the order/wallet components. The first line item drives the display summary.
export function adaptOrder(o: Order, locale: 'en' | 'ar' | 'tr'): OrderView {
  const first = o.items[0]
  const extra = o.items.length > 1 ? ` +${o.items.length - 1}` : ''
  const brand = first ? first.title[locale] || first.title.en : 'Order'
  const fulfill = first ? toFulfillKind(first.fulfillmentType) : 'code'
  const date = fmtDate(o.createdAt)

  const view: OrderView = {
    id: o.id,
    product: first ? `${brand} — ${first.denomination}${extra}` : o.id,
    art: artForCategory(first?.category ?? ''),
    total: o.total,
    method: METHOD_LABEL[o.paymentMethod] ?? o.paymentMethod,
    fulfill,
    status: statusLabel(o.status),
    date,
  }

  if (fulfill === 'code') {
    view.code = o.fulfillment.deliveredCode ?? ''
  } else if (fulfill === 'credit') {
    view.account = first?.playerId || o.fulfillment.creditedToId || '—'
    view.amount = first?.denomination
    view.ts = date
  } else {
    view.ref = o.fulfillment.transferRef
    view.recipient = first?.recipient
    view.steps = o.fulfillment.statusTimeline.map((e) => ({
      k: e.status,
      ts: e.at ? fmtDate(e.at) : '',
    }))
  }

  return view
}
