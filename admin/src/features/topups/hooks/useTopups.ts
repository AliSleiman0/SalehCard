import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { listTopUps, approveTopUp, rejectTopUp, type TopUpListParams } from '../api/topups'

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
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'topups'] }),
  })
}

export function useRejectTopUp() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) => rejectTopUp(id, reason),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'topups'] }),
  })
}
