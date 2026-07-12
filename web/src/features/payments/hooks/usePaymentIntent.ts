import { useQuery } from '@tanstack/react-query'
import { fetchPaymentIntent } from '../api/payments'
import { unwrap } from '@/lib/api-error'
import type { PaymentIntent } from '@/types'

const TERMINAL = ['confirmed', 'expired']

// usePaymentIntent polls one intent every 7s, stopping once it reaches a terminal
// state. Poll limit is 60/min server-side, so 7s is safe. `seed` hydrates the
// query from router state so a fresh redirect renders instantly (hard reload
// falls back to fetching by id).
export function usePaymentIntent(id: string, seed?: PaymentIntent) {
  return useQuery({
    queryKey: ['payment-intent', id],
    queryFn: async () => unwrap(await fetchPaymentIntent(id), 'Failed to load payment'),
    enabled: !!id,
    initialData: seed && seed.id === id ? seed : undefined,
    refetchInterval: (q) => (q.state.data && TERMINAL.includes(q.state.data.status) ? false : 7000),
    refetchIntervalInBackground: false,
  })
}
