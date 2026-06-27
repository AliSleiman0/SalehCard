import { useQuery } from '@tanstack/react-query'
import { listProductCategories } from '../api/categories'

/** Distinct product categories (value + count) for the admin category dropdowns. */
export function useProductCategories() {
  return useQuery({
    queryKey: ['admin', 'products', 'categories'],
    queryFn: () => listProductCategories(),
    staleTime: 60_000,
  })
}
