import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { listOrders, getOrder, type OrderListParams } from '../api/orders'

export function useOrders(params: OrderListParams = {}, opts?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['admin', 'orders', params],
    queryFn: () => listOrders(params),
    // Keep the current page visible while the next page/filter/search loads,
    // instead of flashing the table back to a spinner on every key change.
    placeholderData: keepPreviousData,
    // Cross-domain callers (e.g. the user detail Orders tab) pass enabled:false
    // when the admin lacks orders.view, so the fetch isn't fired to 403.
    enabled: opts?.enabled ?? true,
  })
}

export function useOrder(id: string | undefined) {
  return useQuery({
    queryKey: ['admin', 'order', id],
    queryFn: () => getOrder(id!),
    enabled: !!id,
  })
}
