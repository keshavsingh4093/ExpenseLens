import { createContext } from 'react'
import type { User } from './types'

export type AuthState = {
  user: User | null
  ready: boolean
  setSession: (s: { accessToken: string; user: User }) => void
  setUser: (u: User) => void
  logout: () => Promise<void>
}

export const AuthContext = createContext<AuthState>({
  user: null,
  ready: false,
  setSession: () => {},
  setUser: () => {},
  logout: async () => {},
})
