import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { getSettings, updateSettings, type SettingsInput } from '../api/settings'

export function useSettings() {
  return useQuery({
    queryKey: ['admin', 'settings'],
    queryFn: () => getSettings(),
  })
}

export function useUpdateSettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: SettingsInput) => updateSettings(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'settings'] }),
  })
}
