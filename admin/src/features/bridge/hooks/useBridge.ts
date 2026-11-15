import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from '@/stores/toast'
import i18n from '@/i18n/config'
import {
  listDevices,
  createDevice,
  updateDevice,
  rotateToken,
  deleteDevice,
  checkBalance,
  listCommands,
  retryCommand,
  cancelCommand,
  type CommandListParams,
} from '../api/bridge'

export function useBridgeDevices() {
  return useQuery({
    queryKey: ['admin', 'bridge', 'devices'],
    queryFn: listDevices,
    refetchInterval: 15_000, // surface online/offline + balances without a manual refresh
  })
}

export function useBridgeCommands(params: CommandListParams) {
  return useQuery({
    queryKey: ['admin', 'bridge', 'commands', params],
    queryFn: () => listCommands(params),
    refetchInterval: 10_000,
  })
}

function useInvalidateDevices() {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ queryKey: ['admin', 'bridge', 'devices'] })
}

function useInvalidateCommands() {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ queryKey: ['admin', 'bridge', 'commands'] })
}

export function useCreateDevice() {
  const invalidate = useInvalidateDevices()
  return useMutation({
    mutationFn: ({ name, providers }: { name: string; providers: string[] }) =>
      createDevice(name, providers),
    onSuccess: () => invalidate(),
    onError: () => toast.error(i18n.t('brg_t_register_failed')),
  })
}

export function useUpdateDevice() {
  const invalidate = useInvalidateDevices()
  return useMutation({
    mutationFn: ({ id, patch }: { id: string; patch: { name?: string; enabled?: boolean; providers?: string[] } }) =>
      updateDevice(id, patch),
    onSuccess: () => invalidate(),
    onError: () => toast.error(i18n.t('brg_t_update_failed')),
  })
}

export function useRotateToken() {
  return useMutation({
    mutationFn: (id: string) => rotateToken(id),
    onError: () => toast.error(i18n.t('brg_t_rotate_failed')),
  })
}

export function useDeleteDevice() {
  const invalidate = useInvalidateDevices()
  return useMutation({
    mutationFn: (id: string) => deleteDevice(id),
    onSuccess: () => {
      invalidate()
      toast.success(i18n.t('brg_t_removed'))
    },
    onError: () => toast.error(i18n.t('brg_t_remove_failed')),
  })
}

export function useCheckBalance() {
  return useMutation({
    mutationFn: (id: string) => checkBalance(id),
    onSuccess: () => toast.success(i18n.t('brg_t_balance_queued')),
    onError: () => toast.error(i18n.t('brg_t_balance_failed')),
  })
}

export function useRetryCommand() {
  const invalidate = useInvalidateCommands()
  return useMutation({
    mutationFn: (id: string) => retryCommand(id),
    onSuccess: () => {
      invalidate()
      toast.success(i18n.t('brg_t_requeued'))
    },
    onError: () => toast.error(i18n.t('brg_t_retry_failed')),
  })
}

export function useCancelCommand() {
  const invalidate = useInvalidateCommands()
  return useMutation({
    mutationFn: (id: string) => cancelCommand(id),
    onSuccess: () => {
      invalidate()
      toast.success(i18n.t('brg_t_cancelled'))
    },
    onError: () => toast.error(i18n.t('brg_t_cancel_failed')),
  })
}
