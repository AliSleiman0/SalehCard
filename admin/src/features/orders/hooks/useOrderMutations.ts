import { useMutation, useQueryClient } from '@tanstack/react-query'
import { refundOrder, refundBulkOrders, completeOrder, failOrder } from '../api/orders'

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

export function useRefundBulk() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ ids, reason }: { ids: string[]; reason?: string }) => refundBulkOrders(ids, reason ?? ''),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'orders'] }),
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

export function useFailOrder() {
  const invalidate = useInvalidateOrders()
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason?: string }) => failOrder(id, reason ?? ''),
    onSuccess: (_, { id }) => invalidate(id),
  })
}
