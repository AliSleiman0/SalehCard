import { apiClient } from '@/lib/api-client'
import type { ApiResponse } from '@/types'
import type { AppNotification } from '../types'

export async function fetchNotifications(): Promise<ApiResponse<AppNotification[]>> {
  return apiClient.get<AppNotification[]>('/api/v1/notifications?limit=50')
}

export async function fetchUnreadCount(): Promise<ApiResponse<{ count: number }>> {
  return apiClient.get<{ count: number }>('/api/v1/notifications/unread-count')
}

export async function markAllRead(): Promise<ApiResponse<{ updated: number }>> {
  return apiClient.post<{ updated: number }>('/api/v1/notifications/read-all')
}
