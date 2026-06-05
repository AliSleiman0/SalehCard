import { useMutation } from '@tanstack/react-query'
import { register, type RegisterInput } from '../api/auth'
import { applyAuth } from './applyAuth'
import type { AuthResponse } from '@/types'

export function useRegister() {
  return useMutation<AuthResponse, Error, RegisterInput>({
    mutationFn: async (input) => {
      const res = await register(input)
      if (!res.success || !res.data) {
        throw new Error(res.error?.message ?? 'Registration failed')
      }
      return res.data
    },
    onSuccess: applyAuth,
  })
}
