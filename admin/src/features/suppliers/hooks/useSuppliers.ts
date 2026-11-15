import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from '@/stores/toast'
import {
  listSuppliers,
  getSupplierCatalog,
  syncSupplier,
  importSupplierProducts,
  updateSupplierSettings,
  getSupplierOrders,
  getSyncStatus,
  listSupplierAlerts,
  ackSupplierAlert,
  ackAllSupplierAlerts,
  type ImportItem,
  type SupplierOrdersParams,
  type SupplierSettingsPatch,
} from '../api/suppliers'

/** Supplier list with live balance + health. Polls so balances/health stay fresh. */
export function useSuppliersList() {
  return useQuery({
    queryKey: ['admin', 'suppliers', 'list'],
    queryFn: listSuppliers,
    refetchInterval: 30_000,
  })
}

/** Live upstream catalog for one supplier — fetched only while a browse modal or
 *  picker needs it. Every fetch hits the supplier (no server cache), so pickers
 *  pass a staleTime; `enabled` gates on access (suppliers.view). A failed fetch
 *  isn't retried: the picker falls back to typing the id. */
export function useSupplierCatalog(id: number | null, opts: { enabled?: boolean; staleTime?: number } = {}) {
  return useQuery({
    queryKey: ['admin', 'suppliers', 'catalog', id],
    queryFn: () => getSupplierCatalog(id as number),
    enabled: id != null && (opts.enabled ?? true),
    staleTime: opts.staleTime,
    retry: false,
  })
}

/** The scheduled sync's config + last pass (polled while the page is open). */
export function useSyncStatus() {
  return useQuery({
    queryKey: ['admin', 'suppliers', 'sync-status'],
    queryFn: getSyncStatus,
    refetchInterval: 60_000,
  })
}

/** Supplier alerts (open by default). */
export function useSupplierAlerts(status: 'open' | 'all') {
  return useQuery({
    queryKey: ['admin', 'suppliers', 'alerts', status],
    queryFn: () => listSupplierAlerts(status),
    refetchInterval: 60_000,
  })
}

export function useAckSupplierAlert() {
  const invalidate = useInvalidateSuppliers()
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => ackSupplierAlert(id),
    onSuccess: () => {
      invalidate()
      qc.invalidateQueries({ queryKey: ['admin', 'dashboard'] })
    },
    onError: () => toast.error('Could not acknowledge the alert'),
  })
}

export function useAckAllSupplierAlerts() {
  const invalidate = useInvalidateSuppliers()
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => ackAllSupplierAlerts(),
    onSuccess: () => {
      invalidate()
      qc.invalidateQueries({ queryKey: ['admin', 'dashboard'] })
    },
    onError: () => toast.error('Could not acknowledge the alerts'),
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
    onError: (err) =>
      toast.error(err instanceof Error && err.message ? err.message : "Could not check this supplier's products"),
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
      patch: SupplierSettingsPatch
    }) => updateSupplierSettings(id, patch),
    onSuccess: () => {
      invalidate()
      toast.success('Supplier settings saved')
    },
    onError: () => toast.error('Could not save supplier settings'),
  })
}
