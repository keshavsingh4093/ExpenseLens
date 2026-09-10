import { Link, useNavigate } from 'react-router'
import { Bell, ChevronRight, FileText, LogOut, Settings, ShieldCheck, Tags, Upload, Wallet, type LucideIcon } from 'lucide-react'
import { useAuth } from '@/features/auth/useAuth'
import { useUnreadCount } from '@/features/notifications/useUnreadCount'

const links: { to: string; label: string; icon: LucideIcon; hint: string }[] = [
  { to: '/upload', label: 'Upload receipt', icon: Upload, hint: 'Photo, camera or PDF' },
  { to: '/budgets', label: 'Budgets', icon: Wallet, hint: 'Monthly limits per category' },
  { to: '/categories', label: 'Categories', icon: Tags, hint: 'Defaults and your own' },
  { to: '/warranty', label: 'Warranty', icon: ShieldCheck, hint: 'Track product warranties' },
  { to: '/reports', label: 'Reports', icon: FileText, hint: 'CSV and PDF exports' },
  { to: '/notifications', label: 'Notifications', icon: Bell, hint: 'Alerts and reminders' },
  { to: '/settings', label: 'Settings', icon: Settings, hint: 'Profile, security, preferences' },
]

/** Mobile "More" tab: everything not on the bottom bar. */
export function MorePage() {
  const { user, logout } = useAuth()
  const nav = useNavigate()
  const unread = useUnreadCount()
  const initials = user?.name.split(' ').map((s) => s[0]).join('').slice(0, 2).toUpperCase() ?? ''
  return (
    <div className="mx-auto max-w-3xl">
      <h1 className="mb-4 text-2xl font-semibold">More</h1>
      {user && (
        <Link to="/settings" className="card mb-4 flex items-center gap-3 p-4">
          {user.avatarUrl ? <img src={user.avatarUrl} alt="" className="size-11 rounded-full object-cover" /> : <span className="grid size-11 shrink-0 place-items-center rounded-full bg-brand-100 text-sm font-semibold text-brand-700">{initials}</span>}
          <span className="min-w-0 flex-1">
            <span className="block truncate text-sm font-semibold">{user.name}</span>
            <span className="text-muted block truncate text-xs">{user.email}</span>
          </span>
          <ChevronRight className="text-muted size-4" />
        </Link>
      )}
      <ul className="card divide-y divide-[var(--border)] overflow-hidden">
        {links.map(({ to, label, icon: Icon, hint }) => (
          <li key={to}>
            <Link to={to} className="flex items-center gap-3 px-4 py-3 hover:bg-brand-50/50 dark:hover:bg-brand-900/10">
              <span className="grid size-9 shrink-0 place-items-center rounded-lg bg-brand-50 text-brand-600 dark:bg-brand-900/30"><Icon className="size-4.5" /></span>
              <span className="min-w-0 flex-1">
                <span className="block text-sm font-medium">{label}</span>
                <span className="text-muted block text-xs">{hint}</span>
              </span>
              {to === '/notifications' && unread > 0 && <span className="rounded-full bg-brand-600 px-2 py-0.5 text-xs font-semibold text-white">{unread}</span>}
              <ChevronRight className="text-muted size-4" />
            </Link>
          </li>
        ))}
      </ul>
      <button type="button" onClick={() => logout().then(() => nav('/login', { replace: true }))} className="card mt-4 flex w-full items-center gap-3 px-4 py-3 text-left text-sm font-medium text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-950/30">
        <LogOut className="size-4.5" /> Sign out
      </button>
    </div>
  )
}
