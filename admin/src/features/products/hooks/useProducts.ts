import { keepPreviousData, useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query'
import {
  listProducts,
  getProduct,
  createProduct,
  updateProduct,
  deleteProduct,
  restoreProduct,
  purgeProduct,
  bulkProductAction,
  uploadProductImage,
  listFulfillmentProviders,
  getCategoryProductOrder,
  saveCategoryProductOrder,
  type ProductListParams,
  type ProductInput,
  type BulkAction,
  type BulkFilter,
} from '../api/products'

export function useProducts(params: ProductListParams = {}, opts?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['admin', 'products', params],
    queryFn: () => listProducts(params),
    placeholderData: keepPreviousData,
    // Cross-domain callers (the offer editor's product picker) pass enabled:false
    // when the admin lacks products.view, so the fetch isn't fired to 403.
    enabled: opts?.enabled ?? true,
  })
}

export function useProduct(id: string | undefined) {
  return useQuery({
    queryKey: ['admin', 'product', id],
    queryFn: () => getProduct(id!),
    enabled: !!id,
  })
}

export function useFulfillmentProviders() {
  return useQuery({
    queryKey: ['admin', 'products', 'fulfillment-providers'],
    queryFn: listFulfillmentProviders,
    // Env-driven on the API — changes only on redeploy, so cache generously.
    staleTime: 5 * 60 * 1000,
  })
}

export function useCreateProduct() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ProductInput) => createProduct(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'products'] }),
  })
}

export function useUpdateProduct(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: Partial<ProductInput>) => updateProduct(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'products'] })
      qc.invalidateQueries({ queryKey: ['admin', 'product', id] })
    },
  })
}

/** Cached admin orders carry missingProductIds (which line products no longer
 *  exist); refetch them after a PERMANENT delete so a stale view never links to
 *  it. A trashed product still exists, so its order links stay valid. */
function invalidateOrderViews(qc: QueryClient) {
  qc.invalidateQueries({ queryKey: ['admin', 'orders'] })
  qc.invalidateQueries({ queryKey: ['admin', 'order'] })
}

// Trash mutations move products in/out of every list (and the category counts),
// so they refresh the product lists, the single product, and the tree. A purge
// also removes the product for good, so order views are refreshed too.
function useTrashMutation(fn: (id: string) => Promise<unknown>, opts?: { erases?: boolean }) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: (_res, id) => {
      qc.invalidateQueries({ queryKey: ['admin', 'products'] })
      qc.invalidateQueries({ queryKey: ['admin', 'product', id] })
      qc.invalidateQueries({ queryKey: ['admin', 'categories'] })
      if (opts?.erases) invalidateOrderViews(qc)
    },
  })
}

export function useCategoryProductOrder(categoryId: string) {
  return useQuery({
    queryKey: ['admin', 'products', 'order', categoryId],
    queryFn: () => getCategoryProductOrder(categoryId),
  })
}

export function useSaveCategoryProductOrder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ categoryId, productIds }: { categoryId: string; productIds: string[] }) =>
      saveCategoryProductOrder(categoryId, productIds),
    // Covers the arrange list itself and every product listing (the category
    // panel previews the customer order).
    onSettled: () => qc.invalidateQueries({ queryKey: ['admin', 'products'] }),
  })
}

/** Soft delete (move to trash). */
export function useDeleteProduct() {
  return useTrashMutation(deleteProduct)
}

export function useRestoreProduct() {
  return useTrashMutation(restoreProduct)
}

/** Permanent erase of a trashed product (super admin). */
export function usePurgeProduct() {
  return useTrashMutation(purgeProduct, { erases: true })
}

export function useUploadProductImage() {
  return useMutation({
    mutationFn: (file: File) => uploadProductImage(file),
  })
}

export function useBulkProductAction() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      ids,
      action,
      categoryId,
      filter,
    }: {
      ids: string[]
      action: BulkAction
      categoryId?: string
      filter?: BulkFilter
    }) => bulkProductAction(ids, action, categoryId, filter),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['admin', 'products'] })
      // assign-category moves product counts around the taxonomy tree and the
      // product-category facets — refresh both so the Categories page + filters
      // reflect the reassignment.
      qc.invalidateQueries({ queryKey: ['admin', 'categories'] })
      qc.invalidateQueries({ queryKey: ['admin', 'products', 'categories'] })
      // Only a permanent delete removes products that orders link to.
      if (vars.action === 'purge') invalidateOrderViews(qc)
    },
  })
}
