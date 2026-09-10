import { useAuth as useAuthGeneric } from '@common/auth/useAuth'
import type { Admin } from './types'

export const useAuth = () => useAuthGeneric<Admin>()
