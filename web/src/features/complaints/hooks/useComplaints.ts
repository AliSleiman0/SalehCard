import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createComplaint,
  fetchComplaint,
  fetchComplaints,
  replyComplaint,
  type Complaint,
  type CreateComplaintInput,
} from '../api/complaints'
import type { ApiResponse } from '@/types'

// The storefront has no push, so open complaint pages poll for staff replies.
const POLL_MS = 30_000

function unwrap<T>(res: ApiResponse<T>, fallback: string): T {
  if (!res.success || !res.data) throw new Error(res.error?.message ?? fallback)
  return res.data
}

export function useComplaints() {
  return useQuery({
    queryKey: ['complaints'],
    queryFn: async () => unwrap(await fetchComplaints(), 'Failed to load complaints'),
    refetchInterval: POLL_MS,
  })
}

/** One complaint; `live` pauses polling while the reply box holds a draft. */
export function useComplaint(id: string, live = true) {
  return useQuery({
    queryKey: ['complaints', id],
    queryFn: async () => unwrap(await fetchComplaint(id), 'Failed to load complaint'),
    refetchInterval: live ? POLL_MS : false,
    enabled: !!id,
  })
}

export function useCreateComplaint() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (input: CreateComplaintInput) => unwrap(await createComplaint(input), 'Could not submit'),
    onSuccess: (c: Complaint) => {
      qc.setQueryData(['complaints', c.id], c)
      qc.invalidateQueries({ queryKey: ['complaints'], exact: true })
    },
  })
}

export function useReplyComplaint(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({ body, photoUrl }: { body: string; photoUrl?: string }) =>
      unwrap(await replyComplaint(id, body, photoUrl), 'Could not send'),
    onSuccess: (c: Complaint) => {
      qc.setQueryData(['complaints', id], c)
      qc.invalidateQueries({ queryKey: ['complaints'], exact: true })
    },
  })
}
