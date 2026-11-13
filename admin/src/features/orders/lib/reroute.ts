import type { AdminOrder } from '../api/orders'

/** rerouteState mirrors the server's eligibility rule for "send to another
 *  supplier": a processing supplier (api-mode) order that is flagged stuck, or
 *  parked with no supplier reference. ambiguous = the previous supplier's
 *  result is unknown (a "search:" reference), which needs an explicit OK. */
export function rerouteState(o: AdminOrder): { eligible: boolean; ambiguous: boolean } {
  const api = o.items.some((it) => it.fulfillmentMode === 'api')
  const ref = o.fulfillment.providerRef ?? ''
  const eligible = api && o.status === 'processing' && (!!o.fulfillment.supplierStuckAt || ref === '')
  return { eligible, ambiguous: ref.startsWith('search:') }
}
