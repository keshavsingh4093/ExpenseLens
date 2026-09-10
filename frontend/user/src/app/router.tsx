import { createBrowserRouter, Navigate } from 'react-router'
import { AppShell } from './AppShell'
import { Placeholder } from '@common/ui/Placeholder'
import { RequireAuth, GuestOnly } from '@common/auth/RequireAuth'
import { LoginPage } from '@/features/auth/LoginPage'
import { RegisterPage } from '@/features/auth/RegisterPage'
import { ForgotPasswordPage } from '@/features/auth/ForgotPasswordPage'
import { ResetPasswordPage } from '@/features/auth/ResetPasswordPage'
import { OAuthCallbackPage } from '@/features/auth/OAuthCallbackPage'
import { SettingsLayout } from '@/features/settings/SettingsLayout'
import { ProfilePage } from '@/features/settings/ProfilePage'
import { SecurityPage } from '@/features/settings/SecurityPage'
import { PreferencesPage } from '@/features/settings/PreferencesPage'
import { NotificationsSettingsPage } from '@/features/settings/NotificationsPage'
import { DataPrivacyPage } from '@/features/settings/DataPrivacyPage'
import { ExportPage, HelpPage, AboutPage } from '@/features/settings/StaticPages'
import { UploadPage } from '@/features/upload/UploadPage'
import { DashboardPage } from '@/features/dashboard/DashboardPage'
import { ReviewPage } from '@/features/receipts/ReviewPage'
import { ReceiptsPage } from '@/features/receipts/ReceiptsPage'
import { ReceiptDetailPage } from '@/features/receipts/ReceiptDetailPage'

export const router = createBrowserRouter([
  {
    element: <RequireAuth />,
    children: [
      {
        path: '/',
        element: <AppShell />,
        children: [
          { index: true, element: <DashboardPage /> },
          { path: 'receipts', element: <ReceiptsPage /> },
          { path: 'receipts/:id', element: <ReceiptDetailPage /> },
          { path: 'receipts/:id/review', element: <ReviewPage /> },
          { path: 'upload', element: <UploadPage /> },
          { path: 'analytics', element: <Placeholder title="Analytics" /> },
          { path: 'budgets', element: <Placeholder title="Budgets" /> },
          { path: 'categories', element: <Placeholder title="Categories" /> },
          { path: 'warranty', element: <Placeholder title="Warranty" /> },
          { path: 'reports', element: <Placeholder title="Reports" /> },
          { path: 'notifications', element: <Placeholder title="Notifications" /> },
          { path: 'more', element: <Placeholder title="More" /> },
          {
            path: 'settings',
            element: <SettingsLayout />,
            children: [
              { index: true, element: <ProfilePage /> },
              { path: 'security', element: <SecurityPage /> },
              { path: 'preferences', element: <PreferencesPage /> },
              { path: 'notifications', element: <NotificationsSettingsPage /> },
              { path: 'data', element: <DataPrivacyPage /> },
              { path: 'export', element: <ExportPage /> },
              { path: 'help', element: <HelpPage /> },
              { path: 'about', element: <AboutPage /> },
            ],
          },
        ],
      },
    ],
  },
  {
    element: <GuestOnly />,
    children: [
      { path: '/login', element: <LoginPage /> },
      { path: '/register', element: <RegisterPage /> },
      { path: '/forgot-password', element: <ForgotPasswordPage /> },
      { path: '/reset-password', element: <ResetPasswordPage /> },
    ],
  },
  { path: '/auth/callback', element: <OAuthCallbackPage /> },
  { path: '*', element: <Navigate to="/" replace /> },
])
