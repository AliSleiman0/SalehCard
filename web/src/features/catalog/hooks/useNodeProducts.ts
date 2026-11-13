import { useInfiniteQuery } from '@tanstack/react-query'
import { fetchProducts } from '../api/products'

// Page size for tree-node product lists. The API caps limit at 100.
const PAGE = 60

// useNodeProducts pages through a taxonomy node's products A→Z ("Load more").
// directOnly = only products attached to the node itself (used under a node
// that also has subcategories); otherwise the node's whole subtree.
export function useNodeProducts(categoryId: string | undefined, directOnly: boolean) {
  return useInfiniteQuery({
    queryKey: ['node-products', categoryId, directOnly],
    queryFn: ({ pageParam }) =>
      fetchProducts({ categoryId, directOnly, sort: 'name', limit: PAGE, page: pageParam }),
    initialPageParam: 1,
    getNextPageParam: (last) => {
      const m = last.meta
      return m && m.page < m.pages ? m.page + 1 : undefined
    },
    enabled: !!categoryId,
  })
}
