import { useMutation } from '@tanstack/react-query'
import { validatePromo, type PromoResult } from '../api/promos'
import { ApiError, unwrap } from '@/lib/api-error'

// useValidatePromo previews a promo code (imperative — fired on the Apply click).
export function useValidatePromo() {
  return useMutation<PromoResult, ApiError, { code: string; orderTotal: number }>({
    mutationFn: async (input) => unwrap(await validatePromo(input), 'Invalid code'),
  })
}
