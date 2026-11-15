import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  getComplaint,
  getComplaintCounts,
  listAssignees,
  listComplaints,
  replyComplaint,
  setComplaintAssignee,
  setComplaintStatus,
  type ComplaintDetail,
  type ComplaintListParams,
  type ComplaintStatus,
} from '../api/complaints'
import { LIVE_REFETCH_MS } from '@/lib/query-client'
import { toast } from '@/stores/toast'
import i18n from '@/i18n/config'
import type { ApiResponse } from '@/types'

export function useComplaints(params: ComplaintListParams) {
  return useQuery({
    queryKey: ['admin', 'complaints', 'list', params],
    queryFn: () => listComplaints(params),
    refetchInterval: LIVE_REFETCH_MS,
    placeholderData: keepPreviousData,
  })
}

/** Status tallies for the tabs and the sidebar badge (skipped without complaints.view). */
export function useComplaintCounts(opts?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['admin', 'complaints', 'counts'],
    queryFn: getComplaintCounts,
    refetchInterval: LIVE_REFETCH_MS,
    enabled: opts?.enabled ?? true,
  })
}

export function useAssignees() {
  return useQuery({
    queryKey: ['admin', 'complaints', 'assignees'],
    queryFn: listAssignees,
    staleTime: 5 * 60_000,
  })
}

/**
 * One complaint. `live` turns the 30s poll off while the reply box holds a
 * draft, so a refetch never re-renders the thread under someone mid-typing.
 */
export function useComplaint(id: string | undefined, live: boolean) {
  return useQuery({
    queryKey: ['admin', 'complaints', 'detail', id],
    queryFn: () => getComplaint(id!),
    refetchInterval: live ? LIVE_REFETCH_MS : false,
    enabled: !!id,
  })
}

/** Writes the mutation's returned detail into the cache and refreshes lists/counts. */
function useApplyDetail() {
  const qc = useQueryClient()
  return (res: ApiResponse<ComplaintDetail>) => {
    if (res.data) qc.setQueryData(['admin', 'complaints', 'detail', res.data.id], res)
    qc.invalidateQueries({ queryKey: ['admin', 'complaints', 'list'] })
    qc.invalidateQueries({ queryKey: ['admin', 'complaints', 'counts'] })
  }
}

export function useReplyComplaint(id: string) {
  const apply = useApplyDetail()
  return useMutation({
    mutationFn: ({ body, internal }: { body: string; internal: boolean }) => replyComplaint(id, body, internal),
    onSuccess: (res, vars) => {
      apply(res)
      toast.success(i18n.t(vars.internal ? 'cmp_toast_note_added' : 'cmp_toast_reply_sent'))
    },
  })
}

export function useSetComplaintStatus(id: string) {
  const apply = useApplyDetail()
  return useMutation({
    mutationFn: (status: ComplaintStatus) => setComplaintStatus(id, status),
    onSuccess: (res) => {
      apply(res)
      toast.success(i18n.t('cmp_toast_status_updated'))
    },
    onError: () => toast.error(i18n.t('cmp_toast_status_failed')),
  })
}

export function useSetComplaintAssignee(id: string) {
  const apply = useApplyDetail()
  return useMutation({
    mutationFn: (adminId: string | null) => setComplaintAssignee(id, adminId),
    onSuccess: (res) => {
      apply(res)
      toast.success(i18n.t('cmp_toast_assignee_updated'))
    },
    onError: () => toast.error(i18n.t('cmp_toast_assignee_failed')),
  })
}
