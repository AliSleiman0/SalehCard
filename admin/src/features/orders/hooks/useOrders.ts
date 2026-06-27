import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { listOrders, getOrder, type OrderListParams } from '../api/orders'

export function useOrders(params: OrderListParams = {}) {
  return useQuery({
    queryKey: ['admin', 'orders', params],
    queryFn: () => listOrders(params),
    // Keep the current page visible while the next page/filter/search loads,
    // instead of flashing the table back to a spinner on every key change.
    placeholderData: keepPreviousData,
  })
}

export function useOrder(id: string | undefined) {
  return useQuery({
    queryKey: ['admin', 'order', id],
    queryFn: () => getOrder(id!),
    enabled: !!id,
  })
}
