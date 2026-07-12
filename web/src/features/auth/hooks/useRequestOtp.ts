import { useMutation } from '@tanstack/react-query'
import { requestOtp, type RequestOtpInput } from '../api/auth'
import { ApiError, unwrap } from '@/lib/api-error'

// useRequestOtp fires an SMS code. Surfaces ApiError so callers can branch on
// INVALID_PHONE / OTP_THROTTLED.
export function useRequestOtp() {
  return useMutation<{ sent: boolean }, ApiError, RequestOtpInput>({
    mutationFn: async (input) => unwrap(await requestOtp(input), 'Could not send code'),
  })
}
