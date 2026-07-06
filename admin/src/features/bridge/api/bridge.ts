import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaginationMeta } from '@/types'

/** A registered bridge device (dual-SIM recharge phone). */
export interface BridgeDevice {
  id: string
  name: string
  providers: string[]
  enabled: boolean
  lastSeenAt?: string
  touchBalance?: number
  touchValidity?: string
  alfaBalance?: number
  alfaValidity?: string
  appVersion?: string
  createdAt: string
}

export type BridgeCommandStatus = 'queued' | 'leased' | 'succeeded' | 'failed' | 'cancelled'

/** A recharge command dispatched to a device. Card codes are masked server-side. */
export interface BridgeCommand {
  id: string
  orderId?: string
  provider: string
  type: string
  recipientNumber?: string
  amount?: number
  cardCodeMasked?: string
  status: BridgeCommandStatus
  attempts: number
  statusCode?: number
  rawReply?: string
  failReason?: string
  createdAt: string
  updatedAt: string
}

export interface CommandListParams {
  page?: number
  limit?: number
  status?: BridgeCommandStatus | ''
  provider?: string
  orderId?: string
}

const ADMIN = '/api/admin/bridge'

export function listDevices(): Promise<ApiResponse<BridgeDevice[]>> {
  return apiClient.get<BridgeDevice[]>(`${ADMIN}/devices`)
}

export function createDevice(
  name: string,
  providers: string[],
): Promise<ApiResponse<{ device: BridgeDevice; token: string }>> {
  return apiClient.post<{ device: BridgeDevice; token: string }>(`${ADMIN}/devices`, { name, providers })
}

export function updateDevice(
  id: string,
  patch: { name?: string; enabled?: boolean; providers?: string[] },
): Promise<ApiResponse<BridgeDevice>> {
  return apiClient.patch<BridgeDevice>(`${ADMIN}/devices/${id}`, patch)
}

export function rotateToken(id: string): Promise<ApiResponse<{ token: string }>> {
  return apiClient.post<{ token: string }>(`${ADMIN}/devices/${id}/rotate-token`)
}

export function deleteDevice(id: string): Promise<ApiResponse<unknown>> {
  return apiClient.delete(`${ADMIN}/devices/${id}`)
}

export function checkBalance(id: string): Promise<ApiResponse<{ queued: number }>> {
  return apiClient.post<{ queued: number }>(`${ADMIN}/devices/${id}/check-balance`)
}

export function listCommands(
  params: CommandListParams,
): Promise<ApiResponse<BridgeCommand[]> & { meta?: PaginationMeta }> {
  const q = new URLSearchParams()
  if (params.page !== undefined) q.set('page', String(params.page))
  if (params.status) q.set('status', params.status)
  if (params.provider) q.set('provider', params.provider)
  if (params.orderId) q.set('orderId', params.orderId)
  const qs = q.toString()
  return apiClient.get<BridgeCommand[]>(qs ? `${ADMIN}/commands?${qs}` : `${ADMIN}/commands`)
}

export function retryCommand(id: string): Promise<ApiResponse<BridgeCommand>> {
  return apiClient.post<BridgeCommand>(`${ADMIN}/commands/${id}/retry`)
}

export function cancelCommand(id: string): Promise<ApiResponse<BridgeCommand>> {
  return apiClient.post<BridgeCommand>(`${ADMIN}/commands/${id}/cancel`)
}
