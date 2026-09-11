import { createBrowserRouter, Navigate } from 'react-router'
import { RequireAuth, GuestOnly } from '@common/auth/RequireAuth'
import { AdminShell } from './AdminShell'
import { LoginPage } from '@/features/auth/LoginPage'
import { DashboardPage } from '@/features/dashboard/DashboardPage'
import { UsersPage } from '@/features/users/UsersPage'
import { AdminsPage } from '@/features/admins/AdminsPage'
import { SystemConfigPage } from '@/features/config/SystemConfigPage'
import { ExtractionPage } from '@/features/extraction/ExtractionPage'
import { CategoriesPage } from '@/features/categories/CategoriesPage'
import { AuditPage } from '@/features/audit/AuditPage'

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
          { path: 'admins', element: <AdminsPage /> },
          { path: 'extraction', element: <ExtractionPage /> },
          { path: 'categories', element: <CategoriesPage /> },
          { path: 'system-config', element: <SystemConfigPage /> },
          { path: 'audit', element: <AuditPage /> },
        ],
      },
    ],
  },
  { element: <GuestOnly />, children: [{ path: '/login', element: <LoginPage /> }] },
  { path: '*', element: <Navigate to="/" replace /> },
])
