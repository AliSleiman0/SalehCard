import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaginationMeta } from '@/types'

/** One admin action recorded in the audit log. */
export interface AuditEntry {
  id: string
  actorId: string
  actorEmail: string
  action: string
  targetType: string
  targetId: string
  summary?: Record<string, unknown>
  at: string
}

export interface AuditListParams {
  page?: number
  limit?: number
  action?: string
  targetType?: string
  targetId?: string
  actorId?: string
}

const ADMIN = '/api/admin/audit-log'

export function listAuditLog(
  params: AuditListParams,
): Promise<ApiResponse<AuditEntry[]> & { meta?: PaginationMeta }> {
  const q = new URLSearchParams()
  if (params.page !== undefined) q.set('page', String(params.page))
  if (params.limit !== undefined) q.set('limit', String(params.limit))
  if (params.action) q.set('action', params.action)
  if (params.targetType) q.set('targetType', params.targetType)
  if (params.targetId) q.set('targetId', params.targetId)
  if (params.actorId) q.set('actorId', params.actorId)
  const qs = q.toString()
  return apiClient.get<AuditEntry[]>(qs ? `${ADMIN}?${qs}` : ADMIN)
}
