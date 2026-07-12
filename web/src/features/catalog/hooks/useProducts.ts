import { useQuery } from '@tanstack/react-query'
import { fetchProducts, type ProductsParams } from '../api/products'

export function useProducts(params: ProductsParams = {}, opts?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['products', params],
    queryFn: () => fetchProducts(params),
    enabled: opts?.enabled ?? true,
  })
}
