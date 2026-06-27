import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { listTransactions, getRevenueSummary, type TxListParams } from '../api/finance'

/** Paginated, filterable wallet-ledger feed. */
export function useTransactions(params: TxListParams = {}) {
  return useQuery({
    queryKey: ['admin', 'transactions', params],
    queryFn: () => listTransactions(params),
    placeholderData: keepPreviousData,
  })
}

/** Revenue KPIs + breakdowns + time series for the given range. */
export function useRevenueSummary(range: string) {
  return useQuery({
    queryKey: ['admin', 'revenue-summary', range],
    queryFn: () => getRevenueSummary(range),
    placeholderData: keepPreviousData,
  })
}
