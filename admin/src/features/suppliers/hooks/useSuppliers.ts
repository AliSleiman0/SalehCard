import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from '@/stores/toast'
import {
  listSuppliers,
  getSupplierCatalog,
  syncSupplier,
  importSupplierProducts,
  updateSupplierSettings,
  getSupplierOrders,
  type ImportItem,
  type SupplierOrdersParams,
} from '../api/suppliers'

/** Supplier list with live balance + health. Polls so balances/health stay fresh. */
export function useSuppliersList() {
  return useQuery({
    queryKey: ['admin', 'suppliers', 'list'],
    queryFn: listSuppliers,
    refetchInterval: 30_000,
  })
}

/** Live upstream catalog for one supplier — fetched only while the browse modal is open. */
export function useSupplierCatalog(id: number | null) {
  return useQuery({
    queryKey: ['admin', 'suppliers', 'catalog', id],
    queryFn: () => getSupplierCatalog(id as number),
    enabled: id != null,
  })
}

export function useSupplierOrders(id: number | null, params: SupplierOrdersParams) {
  return useQuery({
    queryKey: ['admin', 'suppliers', 'orders', id, params],
    queryFn: () => getSupplierOrders(id as number, params),
    enabled: id != null,
  })
}

function useInvalidateSuppliers() {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ queryKey: ['admin', 'suppliers'] })
}

export function useSyncSupplier() {
  const invalidate = useInvalidateSuppliers()
  return useMutation({
    mutationFn: (id: number) => syncSupplier(id),
    onSuccess: () => invalidate(),
    onError: () => toast.error('Could not sync supplier catalog'),
  })
}

export function useImportProducts() {
  const invalidate = useInvalidateSuppliers()
  return useMutation({
    mutationFn: ({ id, items }: { id: number; items: ImportItem[] }) =>
      importSupplierProducts(id, items),
    onSuccess: () => invalidate(),
    onError: () => toast.error('Could not import products'),
  })
}

export function useUpdateSupplierSettings() {
  const invalidate = useInvalidateSuppliers()
  return useMutation({
    mutationFn: ({
      id,
      patch,
    }: {
      id: number
      patch: { lowBalanceThreshold?: number | null; markupPercent?: number | null }
    }) => updateSupplierSettings(id, patch),
    onSuccess: () => {
      invalidate()
      toast.success('Supplier settings saved')
    },
    onError: () => toast.error('Could not save supplier settings'),
  })
}
