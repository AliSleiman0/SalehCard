import { useMutation, useQueryClient } from '@tanstack/react-query'
import { submitKyc } from '../api/kyc'
import { ApiError, unwrap } from '@/lib/api-error'
import type { KycProfile, KycSubmissionInput } from '../types'

// useSubmitKyc files the KYC submission and writes the returned profile straight
// into the ['kyc'] cache (submit returns the fresh profile — no refetch needed).
export function useSubmitKyc() {
  const qc = useQueryClient()
  return useMutation<KycProfile, ApiError, KycSubmissionInput>({
    mutationFn: async (input) => unwrap(await submitKyc(input), 'Failed to submit'),
    onSuccess: (profile) => {
      qc.setQueryData(['kyc'], profile)
    },
  })
}
