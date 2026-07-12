import { useMutation } from '@tanstack/react-query'
import { createTopUpIntent } from '../api/payments'
import { ApiError, unwrap } from '@/lib/api-error'
import type { PaymentIntent } from '@/types'

// useCreateTopUpIntent opens an on-chain USDT top-up intent. The caller supplies
// a fresh idempotency key per attempt (held in a ref) so retries don't fork.
export function useCreateTopUpIntent() {
  return useMutation<PaymentIntent, ApiError, { amount: number; network?: string; idempotencyKey: string }>({
    mutationFn: async ({ amount, network, idempotencyKey }) =>
      unwrap(await createTopUpIntent({ amount, network }, idempotencyKey), 'Could not start payment'),
  })
}
