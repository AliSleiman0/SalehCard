import type { AdminOrder, AdminOrderItem } from '../api/orders'

/** rerouteState mirrors the server's eligibility rule (rerouteEligible) for
 *  sending an order to a supplier:
 *  - any processing supplier (api-mode) order ("Send to a supplier"). When the
 *    current supplier may still deliver it, the server asks for an explicit
 *    double-delivery OK (supplier-options `needsAck`);
 *  - fromManual: a processing single-line manual account-credit order that was
 *    never sent to a supplier ("Send via supplier"). Transfers, bridge
 *    recharges, code lines and multi-line orders are excluded.
 *  ambiguous = the previous supplier's result is unknown (a "search:"
 *  reference), which needs an explicit OK. */
export function rerouteState(o: AdminOrder): { eligible: boolean; ambiguous: boolean; fromManual: boolean } {
  const ref = o.fulfillment.providerRef ?? ''
  const ambiguous = ref.startsWith('search:')
  if (o.status !== 'processing') return { eligible: false, ambiguous, fromManual: false }
  if (o.items.some((it) => it.fulfillmentMode === 'api')) {
    return { eligible: true, ambiguous, fromManual: false }
  }
  const f = o.fulfillment
  const manual =
    o.items.length === 1 &&
    isManualCredit(o.items[0]) &&
    !f.supplierAttempt &&
    ref === '' &&
    !(f.supplierAttempts && f.supplierAttempts.length > 0)
  return { eligible: manual, ambiguous, fromManual: manual }
}

function isManualCredit(it: AdminOrderItem): boolean {
  if (it.fulfillmentType !== 'account_credit') return false
  // A legacy line without a stored mode derives manual for account credit.
  return !it.fulfillmentMode || it.fulfillmentMode === 'manual_operator'
}

/** The quantity a supplier is sent for a line: the typed units on a per-unit
 *  line (amount), else the cart qty — mirrors the server's supplierQty. */
export function supplierQty(it: Pick<AdminOrderItem, 'qty' | 'amount'>): number {
  return it.amount != null ? it.amount : it.qty
}

/** Margin of what the customer paid over the supplier cost, in percent; null
 *  when unknown or not comparable (a supplier billing in another currency). */
export function orderMargin(linePrice: number, cost: number | undefined, costCurrency: string | undefined, orderCurrency: string): number | null {
  if (cost == null || !(linePrice > 0)) return null
  if ((costCurrency || 'USD') !== (orderCurrency || 'USD')) return null
  return ((linePrice - cost) / linePrice) * 100
}
