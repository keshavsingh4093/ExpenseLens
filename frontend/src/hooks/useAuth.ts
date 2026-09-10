import { useContext } from 'react'
import { AuthContext } from '@/features/auth/authContext'

export const useAuth = () => useContext(AuthContext)
