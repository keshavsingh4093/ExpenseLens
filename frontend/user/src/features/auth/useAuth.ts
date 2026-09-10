import { useAuth as useAuthGeneric } from '@common/auth/useAuth'
import type { User } from './types'

/** Auth state typed with the user-portal User. */
export const useAuth = () => useAuthGeneric<User>()
