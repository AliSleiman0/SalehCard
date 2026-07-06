import { useQuery } from '@tanstack/react-query'
import { listPayments, type PaymentListParams } from '../api/payments'

export function usePayments(params: PaymentListParams) {
  return useQuery({
    queryKey: ['admin', 'payments', params],
    queryFn: () => listPayments(params),
  })
}
