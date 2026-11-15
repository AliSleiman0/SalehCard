import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { listTopUps, approveTopUp, rejectTopUp, type TopUpListParams } from '../api/topups'
import { toast } from '@/stores/toast'
import i18n from '@/i18n/config'
import { LIVE_REFETCH_MS } from '@/lib/query-client'

export function useTopUps(params: TopUpListParams, opts?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['admin', 'topups', params],
    queryFn: () => listTopUps(params),
    refetchInterval: LIVE_REFETCH_MS,
    // Cross-domain callers (Finance's USDT tab) pass enabled:false when the
    // admin lacks topups.view, so the fetch isn't fired to 403.
    enabled: opts?.enabled ?? true,
  })
}

export function useApproveTopUp() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => approveTopUp(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'topups'] })
      toast.success(i18n.t('top_approved_toast'))
    },
    onError: () => toast.error(i18n.t('top_approve_toast_failed')),
  })
}

export function useRejectTopUp() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) => rejectTopUp(id, reason),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'topups'] })
      toast.info(i18n.t('top_rejected_toast'))
    },
    onError: () => toast.error(i18n.t('top_reject_toast_failed')),
  })
}
