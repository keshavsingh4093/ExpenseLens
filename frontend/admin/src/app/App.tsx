import { QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from 'react-router'
import { queryClient } from '@common/api/queryClient'
import { AuthProvider } from '@common/auth/AuthProvider'
import { ThemeProvider } from '@common/theme/ThemeProvider'
import { adminAuthApi } from '@/features/auth/api'
import type { Admin } from '@/features/auth/types'
import { router } from './router'

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <AuthProvider<Admin> bootstrap={adminAuthApi.refresh} logout={adminAuthApi.logout}>
          <RouterProvider router={router} />
        </AuthProvider>
      </ThemeProvider>
    </QueryClientProvider>
  )
}
