import { useMutation, useQueryClient } from '@tanstack/react-query'
import { refundOrder, completeOrder } from '../api/orders'

/** Invalidates the order detail + list queries after a mutation. */
function useInvalidateOrders() {
  const qc = useQueryClient()
  return (id: string) => {
    qc.invalidateQueries({ queryKey: ['admin', 'order', id] })
    qc.invalidateQueries({ queryKey: ['admin', 'orders'] })
  }
}

export function useRefundOrder() {
  const invalidate = useInvalidateOrders()
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason?: string }) => refundOrder(id, reason ?? ''),
    onSuccess: (_, { id }) => invalidate(id),
  })
}

export function useCompleteOrder() {
  const invalidate = useInvalidateOrders()
  return useMutation({
    mutationFn: ({ id, note, transferRef }: { id: string; note?: string; transferRef?: string }) =>
      completeOrder(id, { note, transferRef }),
    onSuccess: (_, { id }) => invalidate(id),
  })
}
