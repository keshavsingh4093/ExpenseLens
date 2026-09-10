import { QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from 'react-router'
import { queryClient } from '@common/api/queryClient'
import { AuthProvider } from '@common/auth/AuthProvider'
import { ThemeProvider } from '@common/theme/ThemeProvider'
import { authApi } from '@/features/auth/api'
import type { User } from '@/features/auth/types'
import { router } from './router'

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <AuthProvider<User> bootstrap={authApi.refresh} logout={authApi.logout}>
          <RouterProvider router={router} />
        </AuthProvider>
      </ThemeProvider>
    </QueryClientProvider>
  )
}
