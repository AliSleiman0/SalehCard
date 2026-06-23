import { useQuery } from '@tanstack/react-query'
import { fetchCategories, type CategoriesParams } from '../api/categories'

export function useCategories(params: CategoriesParams = {}) {
  return useQuery({
    queryKey: ['categories', params],
    queryFn: () => fetchCategories(params),
    // The taxonomy changes rarely; keep it warm to avoid refetch churn.
    staleTime: 5 * 60 * 1000,
  })
}
