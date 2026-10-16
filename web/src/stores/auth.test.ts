import { describe, it, expect, vi, beforeEach } from 'vitest'
import { queryClient } from '@/lib/query-client'
import { useAuthStore } from './auth'
import type { User } from '@/types'

const user = (id: string) => ({ id, email: `${id}@x.local`, role: 'customer' }) as unknown as User

describe('auth store resets user-scoped cache', () => {
  beforeEach(() => {
    useAuthStore.setState({ user: null, isAuthenticated: false })
  })

  it('resets cached queries when the signed-in user changes', () => {
    const spy = vi.spyOn(queryClient, 'resetQueries').mockResolvedValue()
    useAuthStore.getState().setUser(user('reseller'))
    useAuthStore.getState().setUser(user('customer'))
    expect(spy).toHaveBeenCalledTimes(2)
    spy.mockRestore()
  })

  it('does not reset when the same user is set again (token refresh)', () => {
    useAuthStore.setState({ user: user('a'), isAuthenticated: true })
    const spy = vi.spyOn(queryClient, 'resetQueries').mockResolvedValue()
    useAuthStore.getState().setUser(user('a'))
    expect(spy).not.toHaveBeenCalled()
    spy.mockRestore()
  })

  it('resets on logout only when someone was signed in', () => {
    const spy = vi.spyOn(queryClient, 'resetQueries').mockResolvedValue()
    useAuthStore.getState().logout()
    expect(spy).not.toHaveBeenCalled()
    useAuthStore.setState({ user: user('a'), isAuthenticated: true })
    useAuthStore.getState().logout()
    expect(spy).toHaveBeenCalledTimes(1)
    spy.mockRestore()
  })
})
