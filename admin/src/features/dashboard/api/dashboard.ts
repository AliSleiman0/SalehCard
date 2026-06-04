import { apiClient } from '@/lib/api-client'
import type { ApiResponse, InventoryStats } from '@/types'

export interface DashboardStats {
  // Real (derived from products + codes):
  totalProducts: number
  lowStockCount: number
  codesAvailable: number
  codesDelivered: number
  // Mocked server-side until the orders/wallet modules are wired (see backend TODOs):
  revenueToday: number
  revenueDeltaPct: number
  ordersToday: number
  ordersDeltaPct: number
  activeUsers: number
  walletTopups: number
  pendingTransfers: number
}

export function getDashboardStats(): Promise<ApiResponse<DashboardStats>> {
  return apiClient.get<DashboardStats>('/api/admin/dashboard/stats')
}

export function getLowStock(): Promise<ApiResponse<InventoryStats[]>> {
  return apiClient.get<InventoryStats[]>('/api/admin/dashboard/low-stock')
}
