import type { FulfillmentType } from '@/types'

// FulfillKind is the frontend's compact fulfillment vocabulary used by the
// presentational OrderView / cart. The backend uses 'account_credit' where the
// frontend uses 'credit'; this maps between them. The client never sends a
// fulfillment type (the server derives it), so this is render-only.
export type FulfillKind = 'code' | 'credit' | 'transfer'

export function toFulfillKind(t: FulfillmentType): FulfillKind {
  if (t === 'account_credit') return 'credit'
  if (t === 'transfer') return 'transfer'
  return 'code'
}
