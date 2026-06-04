import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  listInventory,
  listCodes,
  uploadCodes,
  lookupCode,
  setStockThreshold,
  type UploadItem,
} from '../api/codes'
import type { CodeStatus } from '@/types'

export function useInventory() {
  return useQuery({ queryKey: ['admin', 'inventory'], queryFn: () => listInventory() })
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
