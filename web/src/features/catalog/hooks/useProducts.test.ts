import { describe, it, expect, vi } from 'vitest'
import { renderHook } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import React from 'react'
import { useProducts } from './useProducts'

vi.mock('../api/products', () => ({
  fetchProducts: vi.fn(() => new Promise(() => {})),
}))

function makeWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  })

  const Wrapper = ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children)

  return Wrapper
}

describe('useProducts', () => {
  it('returns isLoading=true initially when fetchProducts never resolves', () => {
    const wrapper = makeWrapper()
    const { result } = renderHook(() => useProducts(), { wrapper })
    expect(result.current.isLoading).toBe(true)
  })
})
