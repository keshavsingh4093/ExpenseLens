import { NavLink, Outlet, Link } from 'react-router'
import { Plus, ScanLine } from 'lucide-react'
import { clsx } from 'clsx'
import { sidebarNav, bottomNav } from './nav'

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

function Sidebar() {
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
            <Icon className="size-4.5" />
            {label}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}

type BottomItem = (typeof bottomNav)[number]

function Tab({ item }: { item: BottomItem }) {
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
      <item.icon className="size-5" />
      {item.label}
    </NavLink>
  )
}

function BottomTabs() {
  const [a, b, c, d] = bottomNav
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
      <Tab item={d} />
    </nav>
  )
}
