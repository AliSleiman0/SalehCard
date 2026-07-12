import { useMutation, useQueryClient } from '@tanstack/react-query'
import { submitReview, type SubmitReviewInput } from '../api/reviews'
import { ApiError, unwrap } from '@/lib/api-error'

// useSubmitReview posts a 1–5★ review. On success — and on ALREADY_REVIEWED —
// it invalidates the product's review list + my-review flag so the CTA flips.
export function useSubmitReview() {
  const qc = useQueryClient()
  return useMutation<{ id: string }, ApiError, SubmitReviewInput>({
    mutationFn: async (input) => unwrap(await submitReview(input), 'Failed to submit review'),
    onSuccess: (_data, input) => {
      qc.invalidateQueries({ queryKey: ['my-review', input.productId] })
      qc.invalidateQueries({ queryKey: ['reviews', input.productId] })
    },
    onError: (err, input) => {
      if (err.code === 'ALREADY_REVIEWED') {
        qc.invalidateQueries({ queryKey: ['my-review', input.productId] })
      }
    },
  })
}
