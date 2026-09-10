import { NavLink, Outlet, Link, useNavigate } from 'react-router'
import { Plus, ScanLine, LogOut } from 'lucide-react'
import { useAuth } from '@/features/auth/useAuth'
import { clsx } from 'clsx'
import { sidebarNav, bottomNav } from './nav'
import { useUnreadCount } from '@/features/notifications/useUnreadCount'

export function AppShell() {
  return (
    <div className="min-h-dvh md:flex">
      <Sidebar />
      <main className="flex-1 px-4 pt-4 pb-24 md:px-8 md:py-6 md:pb-8">
        <Outlet />
      </main>
      <BottomTabs />
    </div>
  )
}

function Badge({ count, className }: { count: number; className?: string }) {
  if (count <= 0) return null
  return <span className={clsx('rounded-full bg-brand-600 px-1.5 text-[10px] font-semibold leading-4 text-white', className)} aria-label={`${count} unread`}>{count > 99 ? '99+' : count}</span>
}

function Sidebar() {
  const unread = useUnreadCount()
  return (
    <aside className="sticky top-0 hidden h-dvh w-60 shrink-0 flex-col border-r border-[var(--border)] bg-[var(--surface)] px-4 py-5 md:flex">
      <Link to="/" className="mb-6 flex items-center gap-2 px-2">
        <span className="grid size-8 place-items-center rounded-lg bg-brand-600 text-white">
          <ScanLine className="size-5" />
        </span>
        <span className="text-lg font-semibold">ExpenseLens</span>
      </Link>
      <nav className="flex flex-1 flex-col gap-1">
        {sidebarNav.map(({ to, label, icon: Icon, end }) => (
          <NavLink
            key={to}
            to={to}
            end={end}
            className={({ isActive }) =>
              clsx(
                'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                isActive
                  ? 'bg-brand-600 text-white'
                  : 'text-muted hover:bg-brand-50 hover:text-brand-700 dark:hover:bg-brand-900/30',
              )
            }
          >
            {({ isActive }) => (
              <>
                <Icon className="size-4.5" />
                <span className="flex-1">{label}</span>
                {to === '/notifications' && <Badge count={unread} className={isActive ? 'bg-white text-brand-700' : ''} />}
              </>
            )}
          </NavLink>
        ))}
      </nav>
      <UserBlock />
    </aside>
  )
}

function UserBlock() {
  const { user, logout } = useAuth()
  const nav = useNavigate()
  if (!user) return null
  const initials = user.name.split(' ').map((s) => s[0]).join('').slice(0, 2).toUpperCase()
  return (
    <div className="mt-4 flex items-center gap-3 rounded-lg border border-[var(--border)] p-2">
      {user.avatarUrl ? (
        <img src={user.avatarUrl} alt="" className="size-9 rounded-full object-cover" />
      ) : (
        <div className="grid size-9 shrink-0 place-items-center rounded-full bg-brand-100 text-sm font-semibold text-brand-700">{initials}</div>
      )}
      <div className="min-w-0 flex-1">
        <div className="truncate text-sm font-medium">{user.name}</div>
        <div className="text-muted truncate text-xs">{user.email}</div>
      </div>
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

type BottomItem = (typeof bottomNav)[number]

function Tab({ item, badge }: { item: BottomItem; badge?: number }) {
  return (
    <NavLink
      to={item.to}
      end={item.end}
      className={({ isActive }) =>
        clsx(
          'flex flex-1 flex-col items-center gap-0.5 py-2 text-[11px]',
          isActive ? 'text-brand-600' : 'text-muted',
        )
      }
    >
      <span className="relative">
        <item.icon className="size-5" />
        {!!badge && <Badge count={badge} className="absolute -top-1.5 -right-2.5" />}
      </span>
      {item.label}
    </NavLink>
  )
}

function BottomTabs() {
  const [a, b, c, d] = bottomNav
  const unread = useUnreadCount()
  return (
    <nav className="fixed inset-x-0 bottom-0 z-20 flex items-end border-t border-[var(--border)] bg-[var(--surface)] pb-[env(safe-area-inset-bottom)] md:hidden">
      <Tab item={a} />
      <Tab item={b} />
      <Link
        to="/upload"
        aria-label="Upload receipt"
        className="-mt-5 mx-1 grid size-12 shrink-0 place-items-center rounded-full bg-brand-600 text-white shadow-lg"
      >
        <Plus className="size-6" />
      </Link>
      <Tab item={c} />
      <Tab item={d} badge={unread} />
    </nav>
  )
}
