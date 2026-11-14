import { useState } from 'react'

/**
 * Page state that resets to 1 whenever `key` (a serialised filter set) changes —
 * computed during render, not in an effect, so the first query after a filter
 * change already asks for page 1. (An effect-based reset fires one wasted
 * request for the new filter at the old page, which flashes "no results".)
 */
export function useKeyedPage(key: string): [number, (page: number) => void] {
  const [state, setState] = useState({ key, page: 1 })
  const page = state.key === key ? state.page : 1
  return [page, (p: number) => setState({ key, page: p })]
}
