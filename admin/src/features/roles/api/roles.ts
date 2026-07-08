import { apiClient } from '@/lib/api-client'
import type { ApiResponse } from '@/types'

/** A custom admin role: a named RBAC permission set (backend modules/role). */
export interface AdminRole {
  id: string
  name: string
  description?: string
  permissions: string[]
  createdAt: string
  updatedAt: string
}

/** One grantable permission domain from the backend catalog. */
export interface PermissionDomain {
  key: string
  label: string
}

export interface RoleInput {
  name: string
  description: string
  permissions: string[]
}

// Listing + the catalog are open to every admin (the console needs role names
// to render assignments); create/update/delete are super-admin-only.
export function listRoles(): Promise<ApiResponse<AdminRole[]>> {
  return apiClient.get<AdminRole[]>('/api/admin/roles')
}

export function getPermissionCatalog(): Promise<ApiResponse<{ domains: PermissionDomain[] }>> {
  return apiClient.get<{ domains: PermissionDomain[] }>('/api/admin/roles/permissions')
}

export function createRole(input: RoleInput): Promise<ApiResponse<AdminRole>> {
  return apiClient.post<AdminRole>('/api/admin/roles', input)
}

export function updateRole(id: string, input: RoleInput): Promise<ApiResponse<AdminRole>> {
  return apiClient.put<AdminRole>(`/api/admin/roles/${id}`, input)
}

export function deleteRole(id: string): Promise<ApiResponse<{ deleted: boolean }>> {
  return apiClient.delete<{ deleted: boolean }>(`/api/admin/roles/${id}`)
}
