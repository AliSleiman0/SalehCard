import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  listMethods,
  createMethod,
  updateMethod,
  deleteMethod,
  type MethodInput,
} from '../api/methods'
import { toast } from '@/stores/toast'

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
      toast.success('Payment method created')
    },
    onError: () => toast.error('Could not create payment method'),
  })
}

export function useUpdateMethod() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: MethodInput }) => updateMethod(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: KEY })
      toast.success('Payment method saved')
    },
    onError: () => toast.error('Could not save payment method'),
  })
}

export function useDeleteMethod() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteMethod(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: KEY })
      toast.info('Payment method deleted')
    },
    onError: () => toast.error('Could not delete payment method'),
  })
}
