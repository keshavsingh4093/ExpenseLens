import { NavLink, Outlet, Link, useNavigate } from 'react-router'
import { LayoutDashboard, Users, UserCog, SlidersHorizontal, LogOut, ShieldCheck, Moon, Sun, ScanText, Tags, ScrollText, type LucideIcon } from 'lucide-react'
import { clsx } from 'clsx'
import { useAuth } from '@/features/auth/useAuth'
import { useTheme } from '@common/theme/useTheme'

const nav: { to: string; label: string; icon: LucideIcon; end?: boolean }[] = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard, end: true },
  { to: '/users', label: 'Users', icon: Users },
  { to: '/admins', label: 'Admins', icon: UserCog },
  { to: '/extraction', label: 'Extraction', icon: ScanText },
  { to: '/categories', label: 'Categories', icon: Tags },
  { to: '/system-config', label: 'System Config', icon: SlidersHorizontal },
  { to: '/audit', label: 'Audit log', icon: ScrollText },
]

export function AdminShell() {
  return (
    <div className="min-h-dvh md:flex">
      <Sidebar />
      <MobileBar />
      <main className="flex-1 px-4 pt-4 pb-8 md:px-8 md:py-6">
        <Outlet />
      </main>
    </div>
  )
}

function Brand() {
  return (
    <Link to="/" className="flex items-center gap-2 px-2">
      <span className="grid size-8 place-items-center rounded-lg bg-brand-600 text-white">
        <ShieldCheck className="size-5" />
      </span>
      <span className="text-lg font-semibold">
        ExpenseLens <span className="text-muted text-sm font-normal">Admin</span>
      </span>
    </Link>
  )
}

function NavItems({ className }: { className?: string }) {
  return (
    <nav className={clsx('flex gap-1', className)}>
      {nav.map(({ to, label, icon: Icon, end }) => (
        <NavLink
          key={to}
          to={to}
          end={end}
          className={({ isActive }) =>
            clsx(
              'flex shrink-0 items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium whitespace-nowrap transition-colors',
              isActive ? 'bg-brand-600 text-white' : 'text-muted hover:bg-brand-50 hover:text-brand-700 dark:hover:bg-brand-900/30',
            )
          }
        >
          <Icon className="size-4.5" />
          {label}
        </NavLink>
      ))}
    </nav>
  )
}

function ThemeToggle() {
  const { theme, setTheme } = useTheme()
  const dark = theme === 'dark'
  return (
    <button
      type="button"
      aria-label="Toggle theme"
      onClick={() => setTheme(dark ? 'light' : 'dark')}
      className="text-muted rounded-md p-1.5 hover:bg-brand-50 hover:text-brand-700 dark:hover:bg-brand-900/30"
    >
      {dark ? <Sun className="size-4" /> : <Moon className="size-4" />}
    </button>
  )
}

function UserBlock() {
  const { user, logout } = useAuth()
  const nav = useNavigate()
  if (!user) return null
  return (
    <div className="mt-4 flex items-center gap-2 rounded-lg border border-[var(--border)] p-2">
      <div className="min-w-0 flex-1">
        <div className="truncate text-sm font-medium">{user.name}</div>
        <div className="text-muted truncate text-xs">{user.email} · {user.role}</div>
      </div>
      <ThemeToggle />
      <button
        type="button"
        aria-label="Sign out"
        onClick={() => logout().then(() => nav('/login', { replace: true }))}
        className="text-muted rounded-md p-1.5 hover:bg-brand-50 hover:text-brand-700 dark:hover:bg-brand-900/30"
      >
        <LogOut className="size-4" />
      </button>
    </div>
  )
}

function Sidebar() {
  return (
    <aside className="sticky top-0 hidden h-dvh w-60 shrink-0 flex-col border-r border-[var(--border)] bg-[var(--surface)] px-4 py-5 md:flex">
      <div className="mb-6">
        <Brand />
      </div>
      <NavItems className="flex-1 flex-col" />
      <UserBlock />
    </aside>
  )
}

function MobileBar() {
  const { logout } = useAuth()
  const nav = useNavigate()
  return (
    <div className="sticky top-0 z-20 border-b border-[var(--border)] bg-[var(--surface)] px-3 py-2 md:hidden">
      <div className="flex items-center justify-between">
        <Brand />
        <div className="flex items-center gap-1">
          <ThemeToggle />
          <button
            type="button"
            aria-label="Sign out"
            onClick={() => logout().then(() => nav('/login', { replace: true }))}
            className="text-muted rounded-md p-1.5 hover:bg-brand-50 hover:text-brand-700 dark:hover:bg-brand-900/30"
          >
            <LogOut className="size-4" />
          </button>
        </div>
      </div>
      <NavItems className="mt-2 overflow-x-auto" />
    </div>
  )
}
