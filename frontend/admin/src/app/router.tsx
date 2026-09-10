import { createBrowserRouter, Navigate } from 'react-router'
import { RequireAuth, GuestOnly } from '@common/auth/RequireAuth'
import { AdminShell } from './AdminShell'
import { LoginPage } from '@/features/auth/LoginPage'
import { DashboardPage } from '@/features/dashboard/DashboardPage'
import { UsersPage } from '@/features/users/UsersPage'
import { SystemConfigPage } from '@/features/config/SystemConfigPage'

export const router = createBrowserRouter([
  {
    element: <RequireAuth />,
    children: [
      {
        path: '/',
        element: <AdminShell />,
        children: [
          { index: true, element: <DashboardPage /> },
          { path: 'users', element: <UsersPage /> },
          { path: 'system-config', element: <SystemConfigPage /> },
        ],
      },
    ],
  },
  { element: <GuestOnly />, children: [{ path: '/login', element: <LoginPage /> }] },
  { path: '*', element: <Navigate to="/" replace /> },
])
