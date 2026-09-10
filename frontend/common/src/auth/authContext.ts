import { createContext } from 'react'

export type Session<TUser> = { accessToken: string; user: TUser }

export type AuthState<TUser> = {
  user: TUser | null
  ready: boolean
  setSession: (s: Session<TUser>) => void
  setUser: (u: TUser) => void
  logout: () => Promise<void>
}

export const AuthContext = createContext<AuthState<unknown>>({
  user: null,
  ready: false,
  setSession: () => {},
  setUser: () => {},
  logout: async () => {},
})
