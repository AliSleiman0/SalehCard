import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { listKyc, setKycStatus, type KycListParams, type KycStatus } from '../api/kyc'

export function useKyc(params: KycListParams = {}) {
  return useQuery({
    queryKey: ['admin', 'kyc', params],
    queryFn: () => listKyc(params),
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
