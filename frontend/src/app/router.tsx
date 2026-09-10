import { createBrowserRouter } from 'react-router'
import { AppShell } from './AppShell'
import { Placeholder } from '@/components/ui/Placeholder'

export const router = createBrowserRouter([
  {
    path: '/',
    element: <AppShell />,
    children: [
      { index: true, element: <Placeholder title="Dashboard" /> },
      { path: 'receipts', element: <Placeholder title="Receipts" /> },
      { path: 'receipts/:id', element: <Placeholder title="Receipt Details" /> },
      { path: 'upload', element: <Placeholder title="Upload Receipt" /> },
      { path: 'analytics', element: <Placeholder title="Analytics" /> },
      { path: 'budgets', element: <Placeholder title="Budgets" /> },
      { path: 'categories', element: <Placeholder title="Categories" /> },
      { path: 'warranty', element: <Placeholder title="Warranty" /> },
      { path: 'reports', element: <Placeholder title="Reports" /> },
      { path: 'notifications', element: <Placeholder title="Notifications" /> },
      { path: 'settings/*', element: <Placeholder title="Settings" /> },
    ],
  },
  { path: '/login', element: <Placeholder title="Login" /> },
  { path: '/register', element: <Placeholder title="Register" /> },
])
