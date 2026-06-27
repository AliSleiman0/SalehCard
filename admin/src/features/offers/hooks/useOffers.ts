import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  listOffers,
  getOffer,
  createOffer,
  updateOffer,
  deleteOffer,
  type OfferInput,
  type OfferListParams,
} from '../api/offers'

export function useOffers(params: OfferListParams = {}) {
  return useQuery({
    queryKey: ['admin', 'offers', params],
    queryFn: () => listOffers(params),
    placeholderData: keepPreviousData,
  })
}

export function useOffer(id: string | undefined) {
  return useQuery({
    queryKey: ['admin', 'offer', id],
    queryFn: () => getOffer(id!),
    enabled: !!id,
  })
}

export function useCreateOffer() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: OfferInput) => createOffer(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'offers'] }),
  })
}

export function useUpdateOffer(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: OfferInput) => updateOffer(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'offers'] })
      qc.invalidateQueries({ queryKey: ['admin', 'offer', id] })
    },
  })
}

export function useDeleteOffer() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteOffer(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'offers'] }),
  })
}
