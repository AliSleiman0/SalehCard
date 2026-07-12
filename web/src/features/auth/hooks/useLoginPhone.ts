import { useMutation } from '@tanstack/react-query'
import { loginPhone, type LoginPhoneInput } from '../api/auth'
import { applyAuth, assertCustomerSession } from './applyAuth'
import { ApiError, unwrap } from '@/lib/api-error'
import type { AuthResponse } from '@/types'

// useLoginPhone signs in with phone + password (the non-OTP phone path).
export function useLoginPhone() {
  return useMutation<AuthResponse, ApiError, LoginPhoneInput>({
    mutationFn: async (input) => {
      const data = unwrap(await loginPhone(input), 'Login failed')
      assertCustomerSession(data)
      return data
    },
    onSuccess: applyAuth,
  })
}
