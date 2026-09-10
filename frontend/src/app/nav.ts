import {
  LayoutDashboard,
  Receipt,
  Upload,
  BarChart3,
  Wallet,
  Tags,
  ShieldCheck,
  FileText,
  Bell,
  Settings,
  MoreHorizontal,
  type LucideIcon,
} from 'lucide-react'

export type NavItem = { to: string; label: string; icon: LucideIcon; end?: boolean }

export const sidebarNav: NavItem[] = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard, end: true },
  { to: '/receipts', label: 'Receipts', icon: Receipt },
  { to: '/upload', label: 'Upload Receipt', icon: Upload },
  { to: '/analytics', label: 'Analytics', icon: BarChart3 },
  { to: '/budgets', label: 'Budgets', icon: Wallet },
  { to: '/categories', label: 'Categories', icon: Tags },
  { to: '/warranty', label: 'Warranty', icon: ShieldCheck },
  { to: '/reports', label: 'Reports', icon: FileText },
  { to: '/notifications', label: 'Notifications', icon: Bell },
  { to: '/settings', label: 'Settings', icon: Settings },
]

/** Mobile bottom bar: Dashboard, Receipts, [+], Analytics, More */
export const bottomNav: NavItem[] = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard, end: true },
  { to: '/receipts', label: 'Receipts', icon: Receipt },
  { to: '/analytics', label: 'Analytics', icon: BarChart3 },
  { to: '/more', label: 'More', icon: MoreHorizontal },
]
