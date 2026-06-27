import { relativeTime } from '@/lib/utils'
import type { AdminTx } from '../api/finance'

/** The flat shape the transactions table renders. */
export interface TxView {
  id: string
  user: string
  role: string
  type: string
  amount: number
  /** PayChip key, or '' when there is no payment chip (manual adjustment). */
  method: string
  date: string
  raw: AdminTx
}

/** PayChip keys are wallet/visa/usdt; the API stores card → map it to visa
 *  (mirrors adaptOrder), and 'admin' (manual adjustments) has no chip. */
export function payKey(method: string): string {
  if (method === 'card') return 'visa'
  if (method === 'admin') return ''
  return method
}

/** Map an enriched API transaction to the flat view the finance table renders. */
export function adaptTx(t: AdminTx): TxView {
  return {
    id: t.id,
    user: t.user?.name || 'Unknown',
    role: t.user?.role || '',
    type: t.type,
    amount: t.amount,
    method: payKey(t.method),
    date: relativeTime(t.createdAt),
    raw: t,
  }
}
