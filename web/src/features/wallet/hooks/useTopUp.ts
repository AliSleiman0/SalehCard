import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { topUp, fetchTopUpRequests, type TopUpInput, type TopUpRequest } from '../api/wallet'

export function useTopUp() {
  const qc = useQueryClient()
  return useMutation<TopUpRequest, Error, TopUpInput>({
    mutationFn: async (input) => {
      const res = await topUp(input)
      if (!res.success || !res.data) {
        throw new Error(res.error?.message ?? 'Top-up request failed')
      }
      return res.data
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['topup-requests'] })
    },
  })
}

export function useTopUpRequests() {
  return useQuery({
    queryKey: ['topup-requests'],
    queryFn: async () => {
      const res = await fetchTopUpRequests()
      if (!res.success || !res.data) throw new Error(res.error?.message ?? 'Failed to load')
      return res.data
    },
  })
}
