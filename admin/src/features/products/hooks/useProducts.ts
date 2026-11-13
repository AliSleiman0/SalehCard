import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
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

// Trash mutations move products in/out of every list (and the category counts),
// so they refresh the product lists, the single product, and the tree.
function useTrashMutation(fn: (id: string) => Promise<unknown>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: (_res, id) => {
      qc.invalidateQueries({ queryKey: ['admin', 'products'] })
      qc.invalidateQueries({ queryKey: ['admin', 'product', id] })
      qc.invalidateQueries({ queryKey: ['admin', 'categories'] })
    },
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
  return useTrashMutation(purgeProduct)
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
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'products'] })
      // assign-category moves product counts around the taxonomy tree and the
      // product-category facets — refresh both so the Categories page + filters
      // reflect the reassignment.
      qc.invalidateQueries({ queryKey: ['admin', 'categories'] })
      qc.invalidateQueries({ queryKey: ['admin', 'products', 'categories'] })
    },
  })
}
