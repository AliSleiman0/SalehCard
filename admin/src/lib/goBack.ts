import { useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { firstAllowedRoute } from '@/app/nav'
import { useCan } from '@/stores/auth'

/**
 * Whether the current history entry was created by this app (so the entry
 * before it is also ours and a history back stays on-site).
 *
 * React Router's BrowserRouter stamps every entry it writes with an `idx`:
 * 0 for the entry the tab opened on, +1 per push, unchanged on replace. It lives
 * in `history.state`, so it survives a refresh. idx 0 = opened from a pasted
 * link / new tab / bookmark, where "back" would leave the console.
 */
export function hasInAppHistory(): boolean {
  const s = window.history.state as { idx?: unknown } | null
  return typeof s?.idx === 'number' && s.idx > 0
}

/**
 * Resolves a Back fallback for the current admin: "/" (the Dashboard) becomes
 * their first permitted page when their role can't view the Dashboard, so the
 * fallback never lands on a guard redirect.
 */
export function useBackTarget(fallback: string): string {
  const can = useCan()
  if (fallback === '/' && !can('dashboard.view')) return firstAllowedRoute(can)
  return fallback
}

/**
 * Returns to the previous in-app page, or — with no in-app history — replaces
 * the current entry with `fallback` (the logical parent). Replacing (not pushing)
 * keeps idx unchanged, so Back on the parent then falls back again instead of
 * bouncing to the child or leaving the site. Shared by <BackButton> and the
 * edit pages' post-save / post-delete exits so they all land in the same place.
 */
export function useGoBack(fallback: string): () => void {
  const navigate = useNavigate()
  const target = useBackTarget(fallback)
  return useCallback(() => {
    if (hasInAppHistory()) navigate(-1)
    else navigate(target, { replace: true })
  }, [navigate, target])
}
