import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  listPayments,
  listDeposits,
  attributeDeposit,
  ignoreDeposit,
  type PaymentListParams,
  type DepositListParams,
} from '../api/payments'
import { toast } from '@/stores/toast'
import i18n from '@/i18n/config'

export function usePayments(params: PaymentListParams) {
  return useQuery({
    queryKey: ['admin', 'payments', params],
    queryFn: () => listPayments(params),
  })
}

export function useDeposits(params: DepositListParams, opts?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['admin', 'payments', 'deposits', params],
    queryFn: () => listDeposits(params),
    enabled: opts?.enabled ?? true,
  })
}

export function useAttributeDeposit() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, user, note }: { id: string; user: string; note?: string }) =>
      attributeDeposit(id, { user, note }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'payments'] })
      toast.success(i18n.t('pay_attributed_toast'))
    },
    onError: () => toast.error(i18n.t('pay_attribute_toast_failed')),
  })
}

export function useIgnoreDeposit() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, note }: { id: string; note?: string }) => ignoreDeposit(id, { note }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'payments'] })
      toast.info(i18n.t('pay_ignored_toast'))
    },
    onError: () => toast.error(i18n.t('pay_ignore_toast_failed')),
  })
}
