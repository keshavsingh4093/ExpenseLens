import { Navigate, Outlet, useLocation } from 'react-router'
import { Loader2 } from 'lucide-react'
import { useAuth } from './useAuth'

/** Renders child routes only for signed-in users; otherwise redirects to loginPath. */
export function RequireAuth({ loginPath = '/login' }: { loginPath?: string }) {
  const { user, ready } = useAuth()
  const loc = useLocation()
  if (!ready) {
    return (
      <div className="grid min-h-dvh place-items-center">
        <Loader2 className="size-6 animate-spin text-brand-600" />
      </div>
    )
  }
  if (!user) return <Navigate to={loginPath} state={{ from: loc.pathname }} replace />
  return <Outlet />
}

/** Redirects signed-in users away from auth pages. */
export function GuestOnly({ homePath = '/' }: { homePath?: string }) {
  const { user, ready } = useAuth()
  if (!ready) return null
  if (user) return <Navigate to={homePath} replace />
  return <Outlet />
}
