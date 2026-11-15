import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { LIVE_REFETCH_MS } from '@/lib/live'
import { topUp, fetchTopUpRequests, fetchTopUpMethods, type TopUpInput, type TopUpRequest } from '../api/wallet'

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

/** `live`: re-check every 30s while the page is visible (wallet page). */
export function useTopUpRequests(live = false) {
  return useQuery({
    queryKey: ['topup-requests'],
    queryFn: async () => {
      const res = await fetchTopUpRequests()
      if (!res.success || !res.data) throw new Error(res.error?.message ?? 'Failed to load')
      return res.data
    },
    refetchInterval: live ? LIVE_REFETCH_MS : false,
  })
}

/** Enabled admin-defined top-up methods, in admin sort order. */
export function useTopUpMethods() {
  return useQuery({
    queryKey: ['topup-methods'],
    queryFn: async () => {
      const res = await fetchTopUpMethods()
      if (!res.success || !res.data) throw new Error(res.error?.message ?? 'Failed to load')
      return res.data.filter((m) => m.enabled).sort((a, b) => a.sortOrder - b.sortOrder)
    },
  })
}
