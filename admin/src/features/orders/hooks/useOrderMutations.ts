import { useMutation, useQueryClient } from '@tanstack/react-query'
import { refundOrder, refundBulkOrders, completeOrder, failOrder } from '../api/orders'
import { toast } from '@/stores/toast'
import { ApiError } from '@/lib/api-client'
import i18n from '@/i18n/config'

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
    mutationFn: ({ id, reason, acknowledgeSupplier }: { id: string; reason?: string; acknowledgeSupplier?: boolean }) =>
      refundOrder(id, reason ?? '', acknowledgeSupplier ?? false),
    onSuccess: (_, { id }) => {
      invalidate(id)
      toast.success(i18n.t('ord_t_refunded'))
    },
    // ORDER_AT_SUPPLIER is a confirm step the dialog handles, not a failure.
    onError: (e) => {
      if (!(e instanceof ApiError && e.code === 'ORDER_AT_SUPPLIER')) toast.error(i18n.t('ord_t_refund_failed'))
    },
  })
}

export function useRefundBulk() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ ids, reason }: { ids: string[]; reason?: string }) => refundBulkOrders(ids, reason ?? ''),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'orders'] }),
    onError: () => toast.error(i18n.t('ord_t_bulk_refund_failed')),
  })
}

export function useCompleteOrder() {
  const invalidate = useInvalidateOrders()
  return useMutation({
    mutationFn: ({
      id,
      note,
      transferRef,
      acknowledgeSupplier,
    }: {
      id: string
      note?: string
      transferRef?: string
      acknowledgeSupplier?: boolean
    }) => completeOrder(id, { note, transferRef, acknowledgeSupplier }),
    onSuccess: (_, { id }) => {
      invalidate(id)
      toast.success(i18n.t('ord_t_completed'))
    },
    // ORDER_AT_SUPPLIER is a confirm step the dialog handles, not a failure.
    onError: (e) => {
      if (!(e instanceof ApiError && e.code === 'ORDER_AT_SUPPLIER')) toast.error(i18n.t('ord_t_complete_failed'))
    },
  })
}

export function useFailOrder() {
  const invalidate = useInvalidateOrders()
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason?: string }) => failOrder(id, reason ?? ''),
    onSuccess: (_, { id }) => {
      invalidate(id)
      toast.info(i18n.t('ord_t_marked_failed'))
    },
    onError: () => toast.error(i18n.t('ord_t_mark_failed_failed')),
  })
}
