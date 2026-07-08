import { Navigate } from 'react-router-dom'
import { firstAllowedRoute, itemAllowed } from '@/app/nav'
import { useCan } from '@/stores/auth'

/**
 * Route guard for one RBAC domain (runs inside RequireAdmin, so the session is
 * already established): admins whose role lacks "<domain>.view" are redirected
 * to their first permitted page instead of rendering a page that would 403.
 * With superAdmin, the wildcard permission is required instead (role management).
 */
export function RequireDomain({
  domain,
  superAdmin,
  children,
}: {
  domain?: string
  superAdmin?: boolean
  children: React.ReactNode
}) {
  const can = useCan()
  if (!itemAllowed({ domain, superAdmin }, can)) {
    return <Navigate to={firstAllowedRoute(can)} replace />
  }
  return <>{children}</>
}
