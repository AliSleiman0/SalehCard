import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from '@/stores/toast'
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
    onError: () => toast.error('Could not register device'),
  })
}

export function useUpdateDevice() {
  const invalidate = useInvalidateDevices()
  return useMutation({
    mutationFn: ({ id, patch }: { id: string; patch: { name?: string; enabled?: boolean; providers?: string[] } }) =>
      updateDevice(id, patch),
    onSuccess: () => invalidate(),
    onError: () => toast.error('Could not update device'),
  })
}

export function useRotateToken() {
  return useMutation({
    mutationFn: (id: string) => rotateToken(id),
    onError: () => toast.error('Could not rotate token'),
  })
}

export function useDeleteDevice() {
  const invalidate = useInvalidateDevices()
  return useMutation({
    mutationFn: (id: string) => deleteDevice(id),
    onSuccess: () => {
      invalidate()
      toast.success('Device removed')
    },
    onError: () => toast.error('Could not remove device'),
  })
}

export function useCheckBalance() {
  return useMutation({
    mutationFn: (id: string) => checkBalance(id),
    onSuccess: () => toast.success('Balance check queued'),
    onError: () => toast.error('Could not queue balance check'),
  })
}

export function useRetryCommand() {
  const invalidate = useInvalidateCommands()
  return useMutation({
    mutationFn: (id: string) => retryCommand(id),
    onSuccess: () => {
      invalidate()
      toast.success('Command re-queued')
    },
    onError: () => toast.error('Could not retry command'),
  })
}

export function useCancelCommand() {
  const invalidate = useInvalidateCommands()
  return useMutation({
    mutationFn: (id: string) => cancelCommand(id),
    onSuccess: () => {
      invalidate()
      toast.success('Command cancelled')
    },
    onError: () => toast.error('Could not cancel command'),
  })
}
