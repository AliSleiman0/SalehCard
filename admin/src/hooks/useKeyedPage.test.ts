import { describe, it, expect } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useKeyedPage } from './useKeyedPage'

describe('useKeyedPage', () => {
  it('keeps the page while the key is unchanged and resets to 1 in the same render when it changes', () => {
    const { result, rerender } = renderHook(({ k }) => useKeyedPage(k), { initialProps: { k: 'a' } })
    act(() => result.current[1](5))
    expect(result.current[0]).toBe(5)
    rerender({ k: 'b' })
    expect(result.current[0]).toBe(1) // no intermediate render at page 5
    act(() => result.current[1](3))
    expect(result.current[0]).toBe(3)
  })
})
