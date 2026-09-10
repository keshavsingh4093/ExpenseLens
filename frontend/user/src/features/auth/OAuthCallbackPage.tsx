import { useEffect } from 'react'
import { useNavigate } from 'react-router'
import { Loader2 } from 'lucide-react'
import { useAuth } from '@/features/auth/useAuth'
import { authApi } from './api'

/** The API set the refresh cookie and redirected here; trade it for a session. */
export function OAuthCallbackPage() {
  const { setSession } = useAuth()
  const nav = useNavigate()
  useEffect(() => {
    authApi
      .refresh()
      .then((s) => { setSession(s); nav('/', { replace: true }) })
      .catch(() => nav('/login?error=session', { replace: true }))
  }, [setSession, nav])
  return (
    <div className="grid min-h-dvh place-items-center">
      <Loader2 className="size-6 animate-spin text-brand-600" />
    </div>
  )
}
