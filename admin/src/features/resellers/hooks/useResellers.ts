import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { updateUserRole } from '@/features/users/api/users'
import {
  listResellers,
  getReseller,
  listTiers,
  updateResellerTier,
  adjustResellerBalance,
  createTier,
  updateTier,
  deleteTier,
  type ResellerListParams,
  type BalanceAdjustInput,
  type TierInput,
} from '../api/resellers'

export function useResellers(params: ResellerListParams = {}) {
  return useQuery({
    queryKey: ['admin', 'resellers', params],
    queryFn: () => listResellers(params),
    // Keep the current page visible while the next page/filter/search loads.
    placeholderData: keepPreviousData,
  })
}

export function useReseller(id: string | undefined) {
  return useQuery({
    queryKey: ['admin', 'reseller', id],
    queryFn: () => getReseller(id!),
    enabled: !!id,
  })
}

export function useTiers() {
  return useQuery({
    queryKey: ['admin', 'reseller-tiers'],
    queryFn: () => listTiers(),
  })
}

export function useUpdateResellerTier(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (tier: string) => updateResellerTier(id, tier),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'resellers'] })
      qc.invalidateQueries({ queryKey: ['admin', 'reseller', id] })
      qc.invalidateQueries({ queryKey: ['admin', 'reseller-tiers'] })
    },
  })
}

export function useAdjustResellerBalance(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: BalanceAdjustInput) => adjustResellerBalance(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'resellers'] })
      qc.invalidateQueries({ queryKey: ['admin', 'reseller', id] })
    },
  })
}

export function useCreateTier() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: TierInput) => createTier(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'reseller-tiers'] }),
  })
}

export function useUpdateTier() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: TierInput }) => updateTier(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'reseller-tiers'] })
      qc.invalidateQueries({ queryKey: ['admin', 'resellers'] })
    },
  })
}

export function useDeleteTier() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteTier(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'reseller-tiers'] }),
  })
}

/** Promote a user to the reseller role and assign a starting tier — the
 *  "add reseller" flow (reuses the existing role + tier endpoints). */
export function usePromoteToReseller() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({ userId, tier }: { userId: string; tier: string }) => {
      await updateUserRole(userId, 'reseller')
      if (tier) await updateResellerTier(userId, tier)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'resellers'] })
      qc.invalidateQueries({ queryKey: ['admin', 'reseller-tiers'] })
      qc.invalidateQueries({ queryKey: ['admin', 'users'] })
    },
  })
}
