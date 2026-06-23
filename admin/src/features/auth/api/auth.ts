import { apiClient } from '@/lib/api-client'
import type { ApiResponse, UserRole } from '@/types'

export interface AuthUser {
  id: string
  email: string
  role: UserRole
  name?: string
}

export interface AuthResponse {
  accessToken: string
  user: AuthUser
}

// Real auth against the shared backend (POST /api/v1/auth/*). The access token
// is returned in the body (kept in memory); a rotating refresh token is set as
// an httpOnly cookie so the session can be restored on reload via refresh().
export function login(email: string, password: string): Promise<ApiResponse<AuthResponse>> {
  return apiClient.post<AuthResponse>('/api/v1/auth/login', { email, password })
}

export function refresh(): Promise<ApiResponse<AuthResponse>> {
  return apiClient.post<AuthResponse>('/api/v1/auth/refresh')
}
