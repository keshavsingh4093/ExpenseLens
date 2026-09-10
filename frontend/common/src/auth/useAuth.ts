import { useContext } from 'react'
import { AuthContext, type AuthState } from './authContext'

/** Typed access to the auth state; each portal wraps this with its own user type. */
export const useAuth = <TUser,>() => useContext(AuthContext) as AuthState<TUser>
