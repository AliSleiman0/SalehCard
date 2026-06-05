import { useMutation } from '@tanstack/react-query'
import { login, type LoginInput } from '../api/auth'
import { applyAuth } from './applyAuth'
import type { AuthResponse } from '@/types'

export function useLogin() {
  return useMutation<AuthResponse, Error, LoginInput>({
    mutationFn: async (input) => {
      const res = await login(input)
      if (!res.success || !res.data) {
        throw new Error(res.error?.message ?? 'Login failed')
      }
      return res.data
    },
    onSuccess: applyAuth,
  })
}
