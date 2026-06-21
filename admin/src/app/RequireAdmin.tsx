import { Navigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '@/stores/auth'

/** Route guard: admin app requires an authenticated `admin` user.
 *  Redirects to /login when unauthenticated or lacking the admin role. */
export function RequireAdmin({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isAdmin, hydrated } = useAuthStore()
  const location = useLocation()

  // Wait for the on-load refresh to resolve before deciding — otherwise a
  // reload bounces an authenticated admin to /login before the session restores.
  if (!hydrated) return null

  if (!isAuthenticated || !isAdmin) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }
  return <>{children}</>
}
