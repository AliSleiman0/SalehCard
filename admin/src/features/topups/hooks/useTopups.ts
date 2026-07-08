import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { listTopUps, approveTopUp, rejectTopUp, type TopUpListParams } from '../api/topups'
import { toast } from '@/stores/toast'

export function useTopUps(params: TopUpListParams, opts?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['admin', 'topups', params],
    queryFn: () => listTopUps(params),
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
      toast.success('Top-up approved & credited')
    },
    onError: () => toast.error('Could not approve top-up'),
  })
}

export function useRejectTopUp() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) => rejectTopUp(id, reason),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'topups'] })
      toast.info('Top-up rejected')
    },
    onError: () => toast.error('Could not reject top-up'),
  })
}
