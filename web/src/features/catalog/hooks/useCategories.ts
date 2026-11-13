import { useQuery } from '@tanstack/react-query'
import { fetchCategories, fetchCategoryNode, type CategoriesParams } from '../api/categories'

export function useCategories(params: CategoriesParams = {}, opts?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['categories', params],
    queryFn: () => fetchCategories(params),
    enabled: opts?.enabled ?? true,
    // The taxonomy changes rarely; keep it warm to avoid refetch churn.
    staleTime: 5 * 60 * 1000,
  })
}

// useCategoryNode loads one tree node + its breadcrumb for the drill-down page.
export function useCategoryNode(id: string | undefined) {
  return useQuery({
    queryKey: ['category-node', id],
    queryFn: () => fetchCategoryNode(id!),
    enabled: !!id,
    staleTime: 5 * 60 * 1000,
  })
}
