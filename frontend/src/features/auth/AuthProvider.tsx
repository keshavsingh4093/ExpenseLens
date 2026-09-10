import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { tokenStore } from '@/lib/api/client'
import { queryClient } from '@/lib/api/queryClient'
import { authApi } from './api'
import { AuthContext } from './authContext'
import type { User } from './types'

/** Single-flight bootstrap so StrictMode's double effect (or two mounts) share one refresh call. */
let bootstrap: Promise<User | null> | null = null
function bootstrapSession(): Promise<User | null> {
  if (!bootstrap) {
    bootstrap = authApi
      .refresh()
      .then((s) => {
        tokenStore.set(s.accessToken)
        return s.user
      })
      .catch(() => null)
  }
  return bootstrap
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [ready, setReady] = useState(false)

  useEffect(() => {
    let cancelled = false
    bootstrapSession().then((u) => {
      if (cancelled) return
      if (u) setUser(u)
      setReady(true)
    })
    return () => {
      cancelled = true
    }
  }, [])

  const setSession = useCallback((s: { accessToken: string; user: User }) => {
    tokenStore.set(s.accessToken)
    setUser(s.user)
  }, [])

  const logout = useCallback(async () => {
    try {
      await authApi.logout()
    } finally {
      tokenStore.set(null)
      setUser(null)
      queryClient.clear()
    }
  }, [])

  const value = useMemo(() => ({ user, ready, setSession, setUser, logout }), [user, ready, setSession, logout])
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
