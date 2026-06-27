import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  listPromos,
  getPromo,
  createPromo,
  updatePromo,
  deletePromo,
  type PromoInput,
  type PromoListParams,
} from '../api/promos'

export function usePromos(params: PromoListParams = {}) {
  return useQuery({
    queryKey: ['admin', 'promos', params],
    queryFn: () => listPromos(params),
    placeholderData: keepPreviousData,
  })
}

export function usePromo(id: string | undefined) {
  return useQuery({
    queryKey: ['admin', 'promo', id],
    queryFn: () => getPromo(id!),
    enabled: !!id,
  })
}

export function useCreatePromo() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: PromoInput) => createPromo(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'promos'] }),
  })
}

export function useUpdatePromo(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: PromoInput) => updatePromo(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'promos'] })
      qc.invalidateQueries({ queryKey: ['admin', 'promo', id] })
    },
  })
}

export function useDeletePromo() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deletePromo(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'promos'] }),
  })
}
