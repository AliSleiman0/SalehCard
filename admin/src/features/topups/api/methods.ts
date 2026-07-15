import { apiClient } from '@/lib/api-client'
import type { ApiResponse } from '@/types'

/** Input kind a manual top-up method collects from the customer. */
export type MethodFieldType = 'text' | 'number' | 'select' | 'file'

/** One customer-input spec on a manual top-up method. */
export interface MethodField {
  key: string
  label: string
  type: MethodFieldType
  required: boolean
  options?: string[]
  placeholder?: string
}

/** An admin-defined manual funding method shown on the wallet top-up screen. */
export interface TopUpMethod {
  id: string
  name: string
  instructions: string
  enabled: boolean
  sortOrder: number
  fields: MethodField[]
  createdAt: string
  updatedAt: string
}

/** Create/update payload (server validates + normalizes). */
export interface MethodInput {
  name: string
  instructions: string
  enabled: boolean
  sortOrder: number
  fields: MethodField[]
}

const ADMIN = '/api/admin/wallet/topup-methods'

export function listMethods(): Promise<ApiResponse<TopUpMethod[]>> {
  return apiClient.get<TopUpMethod[]>(ADMIN)
}

export function createMethod(input: MethodInput): Promise<ApiResponse<TopUpMethod>> {
  return apiClient.post<TopUpMethod>(ADMIN, input)
}

export function updateMethod(id: string, input: MethodInput): Promise<ApiResponse<TopUpMethod>> {
  return apiClient.put<TopUpMethod>(`${ADMIN}/${id}`, input)
}

export function deleteMethod(id: string): Promise<ApiResponse<{ deleted: boolean }>> {
  return apiClient.delete<{ deleted: boolean }>(`${ADMIN}/${id}`)
}
