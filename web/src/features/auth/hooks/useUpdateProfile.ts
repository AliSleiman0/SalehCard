import { useMutation } from '@tanstack/react-query'
import { updateProfile, type UpdateProfileInput } from '../api/auth'
import { useAuthStore } from '@/stores/auth'
import type { User } from '@/types'

// useUpdateProfile PATCHes the profile and refreshes the auth store on success.
// The store write is the refresh mechanism — every consumer reads the user from
// the store, so no query-cache invalidation is needed.
export function useUpdateProfile() {
  return useMutation<User, Error, UpdateProfileInput>({
    mutationFn: async (input) => {
      const res = await updateProfile(input)
      if (!res.success || !res.data) {
        throw new Error(res.error?.message ?? 'Failed to update profile')
      }
      return res.data
    },
    onSuccess: (user) => {
      useAuthStore.getState().setUser(user)
    },
  })
}
