import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { tokenStore } from '../api/client'
import { queryClient } from '../api/queryClient'
import { AuthContext, type AuthState, type Session } from './authContext'

/** Single-flight bootstrap so StrictMode's double effect (or two mounts) share one refresh call. */
let bootstrap: Promise<unknown> | null = null
function bootstrapSession<TUser>(fn: () => Promise<Session<TUser>>): Promise<TUser | null> {
  if (!bootstrap) {
    bootstrap = fn()
      .then((s) => {
        tokenStore.set(s.accessToken)
        return s.user
      })
      .catch(() => null)
  }
  return bootstrap as Promise<TUser | null>
}

type Props<TUser> = {
  /** Trades the httpOnly refresh cookie for a session (e.g. POST /openapi/auth/refresh). */
  bootstrap: () => Promise<Session<TUser>>
  /** Revokes the refresh cookie server-side. */
  logout: () => Promise<void>
  children: ReactNode
}

export function AuthProvider<TUser>({ bootstrap: bootstrapFn, logout: logoutFn, children }: Props<TUser>) {
  const [user, setUser] = useState<TUser | null>(null)
  const [ready, setReady] = useState(false)

  useEffect(() => {
    let cancelled = false
    bootstrapSession(bootstrapFn).then((u) => {
      if (cancelled) return
      if (u) setUser(u)
      setReady(true)
    })
    return () => {
      cancelled = true
    }
    // bootstrap runs once per app load
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const setSession = useCallback((s: Session<TUser>) => {
    tokenStore.set(s.accessToken)
    setUser(s.user)
  }, [])

  const logout = useCallback(async () => {
    try {
      await logoutFn()
    } finally {
      tokenStore.set(null)
      setUser(null)
      queryClient.clear()
    }
  }, [logoutFn])

  const value = useMemo(() => ({ user, ready, setSession, setUser, logout }), [user, ready, setSession, logout])
  return <AuthContext.Provider value={value as AuthState<unknown>}>{children}</AuthContext.Provider>
}
