import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  listProducts,
  getProduct,
  createProduct,
  updateProduct,
  deleteProduct,
  bulkProductAction,
  uploadProductImage,
  type ProductListParams,
  type ProductInput,
  type BulkAction,
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

export function useDeleteProduct() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteProduct(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'products'] }),
  })
}

export function useUploadProductImage() {
  return useMutation({
    mutationFn: (file: File) => uploadProductImage(file),
  })
}

export function useBulkProductAction() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ ids, action }: { ids: string[]; action: BulkAction }) => bulkProductAction(ids, action),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'products'] }),
  })
}
