import { artForCategory } from '@/lib/art'
import type { Order, OrderItem, OrderStatus, PaymentMethod } from '@/types'
import type { CodeLine, OrderView } from '../types'
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

// codeLines groups an order's delivered codes per line (one code per unit).
// Old orders carry no items[].codes, only fulfillment.deliveredCode — those
// fall back to a single line titled after the first code item.
export function codeLines(o: Order, locale: 'en' | 'ar' | 'tr'): CodeLine[] {
  const title = (it: OrderItem) => it.title[locale] || it.title.en
  const lines = o.items
    .filter((it) => it.codes && it.codes.length > 0)
    .map((it) => ({ title: title(it), codes: it.codes! }))
  if (lines.length > 0) return lines
  const delivered = o.fulfillment.deliveredCode
  if (!delivered) return []
  const codeItem = o.items.find((it) => toFulfillKind(it.fulfillmentType) === 'code') ?? o.items[0]
  return [{ title: codeItem ? title(codeItem) : '', codes: [delivered] }]
}

// adaptOrder maps a backend Order to the presentational OrderView consumed by
// the order/wallet components. The first line item drives the display summary;
// delivered codes come from every code line (codeLines), and the credit /
// transfer details come from the first manual line — so a mixed order shows
// both its codes and its pending manual lines.
export function adaptOrder(o: Order, locale: 'en' | 'ar' | 'tr'): OrderView {
  const first = o.items[0]
  const extra = o.items.length > 1 ? ` +${o.items.length - 1}` : ''
  const brand = first ? first.title[locale] || first.title.en : 'Order'
  const fulfill = first ? toFulfillKind(first.fulfillmentType) : 'code'
  const date = fmtDate(o.createdAt)
  const lines = codeLines(o, locale)
  const manual = o.items.find((it) => toFulfillKind(it.fulfillmentType) !== 'code')
  const manualKind = manual ? (toFulfillKind(manual.fulfillmentType) as 'credit' | 'transfer') : undefined

  const view: OrderView = {
    id: o.id,
    product: first ? `${brand} — ${first.denomination}${extra}` : o.id,
    art: artForCategory(first?.category ?? ''),
    total: o.total,
    method: METHOD_LABEL[o.paymentMethod] ?? o.paymentMethod,
    fulfill,
    status: statusLabel(o.status),
    date,
    codeLines: lines,
    manualKind,
  }

  if (lines.length > 0 || fulfill === 'code') {
    view.code = lines[0]?.codes[0] ?? o.fulfillment.deliveredCode ?? ''
  }
  if (manualKind === 'credit') {
    view.account = manual?.playerId || o.fulfillment.creditedToId || '—'
    view.amount = manual?.denomination
    view.ts = date
  } else if (manualKind === 'transfer') {
    view.ref = o.fulfillment.transferRef
    view.recipient = manual?.recipient
    view.steps = o.fulfillment.statusTimeline.map((e) => ({
      k: e.status,
      ts: e.at ? fmtDate(e.at) : '',
    }))
  }

  return view
}
