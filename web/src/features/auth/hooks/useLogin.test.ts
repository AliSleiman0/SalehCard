import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import React from 'react'
import { useLogin } from './useLogin'
import * as authApi from '../api/auth'
import { useAuthStore } from '@/stores/auth'
import { setAccessToken } from '@/lib/api-client'

vi.mock('../api/auth', () => ({
  login: vi.fn(),
}))

vi.mock('@/lib/api-client', () => ({
  setAccessToken: vi.fn(),
}))

function makeWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  const Wrapper = ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children)
  return Wrapper
}

const fakeUser = {
  id: 'u1',
  name: 'Test User',
  email: 'a@b.com',
  role: 'customer' as const,
  locale: 'en',
  savedPlayerIds: [],
  walletBalance: 0,
  loyaltyPoints: 0,
  createdAt: '',
  updatedAt: '',
}

describe('useLogin', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useAuthStore.setState({ user: null, isAuthenticated: false })
  })

  it('stores the token and user on success', async () => {
    vi.mocked(authApi.login).mockResolvedValue({
      success: true,
      data: { accessToken: 'tok-123', user: fakeUser },
    })

    const { result } = renderHook(() => useLogin(), { wrapper: makeWrapper() })

    act(() => {
      result.current.mutate({ email: 'a@b.com', password: 'password123' })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(authApi.login).toHaveBeenCalledWith({ email: 'a@b.com', password: 'password123' })
    expect(setAccessToken).toHaveBeenCalledWith('tok-123')
    expect(useAuthStore.getState().user).toEqual(fakeUser)
    expect(useAuthStore.getState().isAuthenticated).toBe(true)
  })

  it('surfaces the API error message and does not authenticate', async () => {
    vi.mocked(authApi.login).mockResolvedValue({
      success: false,
      error: { code: 'UNAUTHORIZED', message: 'invalid credentials' },
    })

    const { result } = renderHook(() => useLogin(), { wrapper: makeWrapper() })

    act(() => {
      result.current.mutate({ email: 'a@b.com', password: 'wrong' })
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.message).toBe('invalid credentials')
    expect(useAuthStore.getState().isAuthenticated).toBe(false)
  })
})
