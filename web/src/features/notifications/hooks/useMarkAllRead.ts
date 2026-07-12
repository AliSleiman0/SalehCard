import { useMutation, useQueryClient } from '@tanstack/react-query'
import { markAllRead } from '../api/notifications'

// useMarkAllRead clears the unread badge and refreshes the inbox. Best-effort —
// the inbox page ignores failures.
export function useMarkAllRead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: markAllRead,
    onSuccess: () => {
      qc.setQueryData(['notifications', 'unread'], 0)
      void qc.invalidateQueries({ queryKey: ['notifications'] })
    },
  })
}
