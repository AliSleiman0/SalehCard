import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  listInventory,
  listCodes,
  listUploadHistory,
  uploadCodes,
  lookupCode,
  setStockThreshold,
  addCode,
  editCode,
  deleteCode,
  expireCode,
  type UploadItem,
} from '../api/codes'
import { toast } from '@/stores/toast'
import i18n from '@/i18n/config'
import { apiErrorText } from '@/lib/apiErrorText'
import type { CodeStatus } from '@/types'

/** Invalidate the inventory list/KPIs and the per-product codes query together
 *  after a single-code mutation. */
function invalidateCodes(qc: ReturnType<typeof useQueryClient>) {
  qc.invalidateQueries({ queryKey: ['admin', 'inventory'] })
  qc.invalidateQueries({ queryKey: ['admin', 'codes'] })
}

/** Full inventory list (all products) — for the thresholds editor and upload picker. */
export function useInventory(enabled = true) {
  return useQuery({ queryKey: ['admin', 'inventory'], queryFn: () => listInventory(), enabled })
}

/** One backend-paginated page of the inventory listing — for the code-stock table. */
export function useInventoryPage(params: { page: number; limit?: number; low?: boolean; below?: boolean; q?: string }) {
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

/** Add a single code to a product's pool. */
export function useAddCode() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ productId, code, pin }: { productId: string; code: string; pin?: string }) =>
      addCode(productId, { code, pin }),
    onSuccess: () => {
      invalidateCodes(qc)
      toast.success(i18n.t('inv_code_added'))
    },
    onError: (e) => toast.error(apiErrorText(e, 'inv_add_failed')),
  })
}

/** Edit an available code's value/pin. */
export function useEditCode() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ productId, codeId, code, pin }: { productId: string; codeId: string; code: string; pin?: string }) =>
      editCode(productId, codeId, { code, pin }),
    onSuccess: () => {
      invalidateCodes(qc)
      toast.success(i18n.t('inv_code_updated'))
    },
    onError: (e) => toast.error(apiErrorText(e, 'inv_update_failed')),
  })
}

/** Delete a non-delivered code. */
export function useDeleteCode() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ productId, codeId }: { productId: string; codeId: string }) => deleteCode(productId, codeId),
    onSuccess: () => {
      invalidateCodes(qc)
      toast.success(i18n.t('inv_code_deleted'))
    },
    onError: (e) => toast.error(apiErrorText(e, 'inv_delete_failed')),
  })
}

/** Expire (retire) an available code. */
export function useExpireCode() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ code }: { code: string; productId: string }) => expireCode(code),
    onSuccess: () => {
      invalidateCodes(qc)
      toast.success(i18n.t('inv_code_expired'))
    },
    onError: (e) => toast.error(apiErrorText(e, 'inv_expire_failed')),
  })
}
