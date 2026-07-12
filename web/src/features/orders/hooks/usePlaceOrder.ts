import { useMutation, useQueryClient } from '@tanstack/react-query'
import { placeOrder } from '../api/orders'
import type { PlacedOrder, PlaceOrderInput } from '@/types'

export interface PlaceOrderArgs {
  input: PlaceOrderInput
  idempotencyKey: string
}

/** Order failure carrying the API's machine code (e.g. KYC_REQUIRED). */
export class OrderError extends Error {
  constructor(
    message: string,
    readonly code?: string,
  ) {
    super(message)
  }
}

// usePlaceOrder submits a checkout. On success it invalidates the orders and
// wallet caches so history and balance reflect the new order.
export function usePlaceOrder() {
  const qc = useQueryClient()
  return useMutation<PlacedOrder, Error, PlaceOrderArgs>({
    mutationFn: async ({ input, idempotencyKey }) => {
      const res = await placeOrder(input, idempotencyKey)
      if (!res.success || !res.data) {
        throw new OrderError(res.error?.message ?? 'Order failed', res.error?.code)
      }
      return res.data
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['orders'] })
      void qc.invalidateQueries({ queryKey: ['wallet'] })
    },
  })
}
