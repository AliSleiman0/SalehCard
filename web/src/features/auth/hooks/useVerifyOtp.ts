import { useMutation } from '@tanstack/react-query'
import { verifyOtp, type VerifyOtpInput } from '../api/auth'
import { applyAuth, assertCustomerSession } from './applyAuth'
import { ApiError, unwrap } from '@/lib/api-error'
import type { AuthResponse } from '@/types'

// useVerifyOtp completes a code login (or account creation on first sign-in) and
// applies the resulting session. Surfaces OTP_INVALID/OTP_EXPIRED/OTP_LOCKED/
// WEAK_PASSWORD as ApiError codes.
export function useVerifyOtp() {
  return useMutation<AuthResponse, ApiError, VerifyOtpInput>({
    mutationFn: async (input) => {
      const data = unwrap(await verifyOtp(input), 'Verification failed')
      assertCustomerSession(data)
      return data
    },
    onSuccess: applyAuth,
  })
}
