import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  listInventory,
  listCodes,
  listUploadHistory,
  uploadCodes,
  lookupCode,
  setStockThreshold,
  type UploadItem,
} from '../api/codes'
import type { CodeStatus } from '@/types'

/** Full inventory list (all products) — for the thresholds editor and upload picker. */
export function useInventory(enabled = true) {
  return useQuery({ queryKey: ['admin', 'inventory'], queryFn: () => listInventory(), enabled })
}

/** One backend-paginated page of the inventory listing — for the code-stock table. */
export function useInventoryPage(params: { page: number; limit?: number; low?: boolean }) {
  return useQuery({
    queryKey: ['admin', 'inventory', 'page', params],
    queryFn: () => listInventory(params),
    // Keep the current page visible while the next page/filter loads.
    placeholderData: keepPreviousData,
  })
}

export function useUploadHistory() {
  return useQuery({ queryKey: ['admin', 'upload-history'], queryFn: () => listUploadHistory() })
}

export function useCodes(
  productId: string | undefined,
  params: { status?: CodeStatus; page?: number; limit?: number } = {}
) {
  return useQuery({
    queryKey: ['admin', 'codes', productId, params],
    queryFn: () => listCodes(productId!, params),
    enabled: !!productId,
  })
}

export function useUploadCodes() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ productId, codes }: { productId: string; codes: UploadItem[] }) =>
      uploadCodes(productId, codes),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'inventory'] })
      qc.invalidateQueries({ queryKey: ['admin', 'codes'] })
      qc.invalidateQueries({ queryKey: ['admin', 'upload-history'] })
    },
  })
}

export function useLookupCode(code: string) {
  return useQuery({
    queryKey: ['admin', 'code-audit', code],
    queryFn: () => lookupCode(code),
    enabled: code.trim().length >= 3,
  })
}

export function useSetThreshold() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ productId, threshold }: { productId: string; threshold: number }) =>
      setStockThreshold(productId, threshold),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'inventory'] }),
  })
}
