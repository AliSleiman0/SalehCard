import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { listTopUps, approveTopUp, rejectTopUp, type TopUpListParams } from '../api/topups'
import { toast } from '@/stores/toast'

export function useTopUps(params: TopUpListParams) {
  return useQuery({
    queryKey: ['admin', 'topups', params],
    queryFn: () => listTopUps(params),
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
