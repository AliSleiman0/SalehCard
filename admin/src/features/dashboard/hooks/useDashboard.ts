import { useQuery } from '@tanstack/react-query'
import { getDashboardStats, getLowStock, getRevenueChart, getFulfillmentBreakdown } from '../api/dashboard'

export function useDashboardStats() {
  return useQuery({ queryKey: ['admin', 'dashboard', 'stats'], queryFn: () => getDashboardStats() })
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
