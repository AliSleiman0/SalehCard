import { apiClient } from '@/lib/api-client'
import type { ApiResponse, InventoryStats } from '@/types'

export interface DashboardStats {
  // Derived from products + codes:
  totalProducts: number
  lowStockCount: number
  codesAvailable: number
  codesDelivered: number
  // Derived from orders:
  revenueToday: number
  revenueDeltaPct: number
  ordersToday: number
  ordersDeltaPct: number
  pendingTransfers: number
  revenueSpark: number[] // last-14-day daily revenue
  ordersSpark: number[] // last-14-day daily order count
  // Derived from users / wallet ledger:
  activeUsers: number // accounts seen in the last 24h
  walletTopups: number // top-ups credited today
}

export function getDashboardStats(): Promise<ApiResponse<DashboardStats>> {
  return apiClient.get<DashboardStats>('/api/admin/dashboard/stats')
}

// API base, mirroring api-client's resolution. Used for the unauthenticated
// /health probe, which sits at the server root (not under /api) and does NOT use
// the standard { success, data } envelope.
const API_BASE = (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? 'http://localhost:8080'

export interface HealthStatus {
  api: boolean // the API process answered at all
  db: boolean // GET /health reported the database reachable
}

// getHealth probes GET /health and classifies the outcome instead of throwing:
// 200 {status:"ok"} → API + DB up; 503 → API up but DB down; network error → API down.
export async function getHealth(): Promise<HealthStatus> {
  try {
    const res = await fetch(`${API_BASE}/health`, { credentials: 'include' })
    if (!res.ok) return { api: true, db: false }
    const body = (await res.json().catch(() => ({}))) as { status?: string }
    return { api: true, db: body.status === 'ok' }
  } catch {
    return { api: false, db: false }
  }
}

export function getLowStock(): Promise<ApiResponse<InventoryStats[]>> {
  return apiClient.get<InventoryStats[]>('/api/admin/dashboard/low-stock')
}

export interface RevenueChart {
  range: string
  labels: string[]
  series: number[]
}

export function getRevenueChart(range: string): Promise<ApiResponse<RevenueChart>> {
  return apiClient.get<RevenueChart>(`/api/admin/dashboard/revenue-chart?range=${encodeURIComponent(range)}`)
}

export interface FulfillmentSlice {
  key: string
  label: string
  value: number
}

export function getFulfillmentBreakdown(): Promise<ApiResponse<FulfillmentSlice[]>> {
  return apiClient.get<FulfillmentSlice[]>('/api/admin/dashboard/fulfillment-breakdown')
}
