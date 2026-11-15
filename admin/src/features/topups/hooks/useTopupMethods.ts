import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  listMethods,
  createMethod,
  updateMethod,
  deleteMethod,
  type MethodInput,
} from '../api/methods'
import { toast } from '@/stores/toast'
import i18n from '@/i18n/config'

const KEY = ['admin', 'topup-methods']

export function useTopupMethods(opts?: { enabled?: boolean }) {
  return useQuery({
    queryKey: KEY,
    queryFn: () => listMethods(),
    enabled: opts?.enabled ?? true,
  })
}

export function useCreateMethod() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: MethodInput) => createMethod(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: KEY })
      toast.success(i18n.t('top_method_created'))
    },
    onError: () => toast.error(i18n.t('top_method_create_failed')),
  })
}

export function useUpdateMethod() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: MethodInput }) => updateMethod(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: KEY })
      toast.success(i18n.t('top_method_saved'))
    },
    onError: () => toast.error(i18n.t('top_method_save_toast_failed')),
  })
}

export function useDeleteMethod() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteMethod(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: KEY })
      toast.info(i18n.t('top_method_deleted'))
    },
    onError: () => toast.error(i18n.t('top_method_delete_failed')),
  })
}
