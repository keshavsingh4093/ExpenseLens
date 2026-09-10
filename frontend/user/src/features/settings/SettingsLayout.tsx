import { NavLink, Outlet } from 'react-router'
import { clsx } from 'clsx'
import { User, Shield, SlidersHorizontal, Bell, Database, Download, HelpCircle, Info, type LucideIcon } from 'lucide-react'

const items: { to: string; label: string; icon: LucideIcon; end?: boolean }[] = [
  { to: '/settings', label: 'Profile', icon: User, end: true },
  { to: '/settings/security', label: 'Security', icon: Shield },
  { to: '/settings/preferences', label: 'Preferences', icon: SlidersHorizontal },
  { to: '/settings/notifications', label: 'Notifications', icon: Bell },
  { to: '/settings/data', label: 'Data & Privacy', icon: Database },
  { to: '/settings/export', label: 'Export Data', icon: Download },
  { to: '/settings/help', label: 'Help & Support', icon: HelpCircle },
  { to: '/settings/about', label: 'About', icon: Info },
]

export function SettingsLayout() {
  return (
    <div>
      <h1 className="mb-4 text-2xl font-semibold">Settings</h1>
      <div className="card grid overflow-hidden md:grid-cols-[220px_1fr]">
        <nav className="flex gap-1 overflow-x-auto border-b border-[var(--border)] p-2 md:flex-col md:border-r md:border-b-0">
          {items.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                clsx(
                  'flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-sm whitespace-nowrap',
                  isActive ? 'bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300' : 'text-muted hover:bg-brand-50/60 dark:hover:bg-brand-900/20',
                )
              }
            >
              <Icon className="size-4" /> {label}
            </NavLink>
          ))}
        </nav>
        <div className="p-5 md:p-6">
          <Outlet />
        </div>
      </div>
    </div>
  )
}

export function SettingsSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="max-w-xl">
      <h2 className="mb-4 text-base font-semibold">{title}</h2>
      {children}
    </section>
  )
}
