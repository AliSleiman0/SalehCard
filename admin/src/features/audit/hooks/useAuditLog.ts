import { useQuery } from '@tanstack/react-query'
import { listAuditLog, type AuditListParams } from '../api/audit'

export function useAuditLog(params: AuditListParams) {
  return useQuery({
    queryKey: ['admin', 'audit', params],
    queryFn: () => listAuditLog(params),
  })
}
