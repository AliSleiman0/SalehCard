import { useMutation, useQueryClient } from '@tanstack/react-query'
import { placeOrder } from '../api/orders'
import type { Order, PlaceOrderInput } from '@/types'

export interface PlaceOrderArgs {
  input: PlaceOrderInput
  idempotencyKey: string
}

// usePlaceOrder submits a checkout. On success it invalidates the orders and
// wallet caches so history and balance reflect the new order.
export function usePlaceOrder() {
  const qc = useQueryClient()
  return useMutation<Order, Error, PlaceOrderArgs>({
    mutationFn: async ({ input, idempotencyKey }) => {
      const res = await placeOrder(input, idempotencyKey)
      if (!res.success || !res.data) {
        throw new Error(res.error?.message ?? 'Order failed')
      }
      return res.data
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['orders'] })
      void qc.invalidateQueries({ queryKey: ['wallet'] })
    },
  })
}
