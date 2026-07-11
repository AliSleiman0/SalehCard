import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { useAuthStore } from '@/stores/auth'

const hydrating = (
  <div className="flex items-center justify-center h-screen">Loading...</div>
)

// RequireAuth gates customer pages. While the on-load session restore is in
// flight it renders a placeholder (rather than flashing /login); once hydrated,
// unauthenticated users are redirected to /login.
export function RequireAuth({ children }: { children: ReactNode }) {
  const hydrated = useAuthStore((s) => s.hydrated)
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)

  if (!hydrated) return hydrating
  if (!isAuthenticated) return <Navigate to="/login" replace />
  return <>{children}</>
}

// RequireReseller gates the agent dashboard: RequireAuth plus a role check —
// only accounts an admin promoted to role=reseller may enter; everyone else is
// sent to their normal dashboard. (setUser stores user and isAuthenticated
// together, so user is always populated once hydrated+authenticated.)
export function RequireReseller({ children }: { children: ReactNode }) {
  const hydrated = useAuthStore((s) => s.hydrated)
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const user = useAuthStore((s) => s.user)

  if (!hydrated) return hydrating
  if (!isAuthenticated) return <Navigate to="/login" replace />
  if (user?.role !== 'reseller') return <Navigate to="/dashboard" replace />
  return <>{children}</>
}

// RedirectIfAuthed sends already-authenticated users away from /login & /register.
export function RedirectIfAuthed({ children }: { children: ReactNode }) {
  const hydrated = useAuthStore((s) => s.hydrated)
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)

  if (!hydrated) return hydrating
  if (isAuthenticated) return <Navigate to="/dashboard" replace />
  return <>{children}</>
}
