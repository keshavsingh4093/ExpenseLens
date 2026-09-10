import { Navigate, Outlet, useLocation } from 'react-router'
import { useAuth } from '@/hooks/useAuth'
import { Loader2 } from 'lucide-react'

export function RequireAuth() {
  const { user, ready } = useAuth()
  const loc = useLocation()
  if (!ready) {
    return (
      <div className="grid min-h-dvh place-items-center">
        <Loader2 className="size-6 animate-spin text-brand-600" />
      </div>
    )
  }
  if (!user) return <Navigate to="/login" state={{ from: loc.pathname }} replace />
  return <Outlet />
}

/** Redirects signed-in users away from auth pages. */
export function GuestOnly() {
  const { user, ready } = useAuth()
  if (!ready) return null
  if (user) return <Navigate to="/" replace />
  return <Outlet />
}
