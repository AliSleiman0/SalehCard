import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { listKyc, setKycStatus, type KycListParams, type KycStatus } from '../api/kyc'
import { LIVE_REFETCH_MS } from '@/lib/query-client'

export function useKyc(params: KycListParams = {}) {
  return useQuery({
    queryKey: ['admin', 'kyc', params],
    queryFn: () => listKyc(params),
    refetchInterval: LIVE_REFETCH_MS,
    placeholderData: keepPreviousData,
  })
}

export function useSetKycStatus() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, status, reason }: { id: string; status: KycStatus; reason?: string }) =>
      setKycStatus(id, status, reason ?? ''),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'kyc'] }),
  })
}
