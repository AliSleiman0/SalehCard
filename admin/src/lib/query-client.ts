import { QueryClient } from '@tanstack/react-query'

// LIVE_REFETCH_MS is the auto-refresh cadence for the operational queues
// (orders, top-ups, KYC, users, dashboard panels) so admins never sit on a
// stale list until they reload (client msg #90). React Query pauses interval
// refetches while the tab is hidden, so idle background tabs cost nothing.
// Editor pages are deliberately NOT polled: they seed form state from their
// query, and a refetch carrying someone else's change would clobber edits.
export const LIVE_REFETCH_MS = 30_000

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 300_000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
})
