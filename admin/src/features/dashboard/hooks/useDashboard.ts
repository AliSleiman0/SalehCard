import { useQuery } from '@tanstack/react-query'
import { getDashboardStats, getLowStock, getRevenueChart, getFulfillmentBreakdown, getHealth } from '../api/dashboard'

export function useDashboardStats(opts?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['admin', 'dashboard', 'stats'],
    queryFn: () => getDashboardStats(),
    enabled: opts?.enabled ?? true,
    refetchInterval: 5_000, // live KPI tiles; pauses when tab hidden (RQ default)
  })
}

export function useLowStock() {
  return useQuery({ queryKey: ['admin', 'dashboard', 'low-stock'], queryFn: () => getLowStock() })
}

export function useRevenueChart(range: string) {
  return useQuery({ queryKey: ['admin', 'dashboard', 'revenue-chart', range], queryFn: () => getRevenueChart(range) })
}

export function useFulfillmentBreakdown() {
  return useQuery({ queryKey: ['admin', 'dashboard', 'fulfillment-breakdown'], queryFn: () => getFulfillmentBreakdown() })
}

export function useHealth() {
  return useQuery({
    queryKey: ['admin', 'dashboard', 'health'],
    queryFn: () => getHealth(),
    refetchInterval: 30_000, // live-ish system status
    staleTime: 15_000,
  })
}
