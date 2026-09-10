import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Search, Users } from 'lucide-react'
import { PageHeader } from '@common/ui/PageHeader'
import { Input } from '@common/ui/Field'
import { Button } from '@common/ui/Button'
import { Skeleton } from '@common/ui/Skeleton'
import { EmptyState } from '@common/ui/EmptyState'
import { Modal } from '@common/ui/Modal'
import { Alert } from '@common/ui/Alert'
import { useDebounce } from '@common/hooks/useDebounce'
import { formatDate } from '@common/format/date'
import { usersApi, userKeys, type PortalUser } from './api'

const PAGE = 20

export function UsersPage() {
  const [q, setQ] = useState('')
  const [page, setPage] = useState(1)
  const [selected, setSelected] = useState<PortalUser | null>(null)
  const dq = useDebounce(q, 300)
  const params = { q: dq, page, pageSize: PAGE }
  const list = useQuery({ queryKey: userKeys.list(params), queryFn: () => usersApi.list(params), placeholderData: (prev) => prev })
  const pages = Math.max(1, Math.ceil((list.data?.total ?? 0) / PAGE))

  return (
    <div>
      <PageHeader title="Users" subtitle={list.data ? `${list.data.total.toLocaleString('en-IN')} accounts` : undefined} />
      <div className="relative mb-4 max-w-md">
        <Search className="text-muted pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2" />
        <Input
          placeholder="Search by email or name"
          value={q}
          onChange={(e) => {
            setQ(e.target.value)
            setPage(1)
          }}
          className="pl-9"
          aria-label="Search users"
        />
      </div>
      {list.error && <Alert>Could not load users.</Alert>}
      <div className="card overflow-x-auto">
        {list.isLoading ? (
          <div className="grid gap-2 p-4">{Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-10" />)}</div>
        ) : !list.data?.users.length ? (
          <EmptyState icon={Users} title="No users found" description={dq ? 'Try a different search.' : 'Nobody has signed up yet.'} />
        ) : (
          <table className="w-full min-w-[640px] text-sm">
            <thead className="text-muted text-left text-xs uppercase">
              <tr className="border-b border-[var(--border)]">
                <th className="px-4 py-3 font-medium">User</th>
                <th className="px-4 py-3 font-medium">Sign-in</th>
                <th className="px-4 py-3 font-medium">Currency</th>
                <th className="px-4 py-3 font-medium">Joined</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[var(--border)]">
              {list.data.users.map((u) => (
                <tr key={u.id} className="cursor-pointer hover:bg-brand-50/50 dark:hover:bg-brand-900/10" onClick={() => setSelected(u)}>
                  <td className="px-4 py-3">
                    <div className="font-medium">{u.name}</div>
                    <div className="text-muted text-xs">{u.email}</div>
                  </td>
                  <td className="px-4 py-3">{[u.hasPassword ? 'password' : null, ...u.providers].filter(Boolean).join(', ') || '—'}</td>
                  <td className="px-4 py-3">{u.preferences.currency}</td>
                  <td className="px-4 py-3">{formatDate(u.createdAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
      {pages > 1 && (
        <div className="mt-3 flex items-center justify-between text-sm">
          <span className="text-muted">Page {page} of {pages}</span>
          <div className="flex gap-2">
            <Button variant="secondary" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>Previous</Button>
            <Button variant="secondary" size="sm" disabled={page >= pages} onClick={() => setPage((p) => p + 1)}>Next</Button>
          </div>
        </div>
      )}
      <UserDetail user={selected} onClose={() => setSelected(null)} />
    </div>
  )
}

function UserDetail({ user, onClose }: { user: PortalUser | null; onClose: () => void }) {
  const q = useQuery({ queryKey: userKeys.detail(user?.id ?? ''), queryFn: () => usersApi.get(user!.id), enabled: !!user })
  return (
    <Modal open={!!user} onClose={onClose} title={user?.name ?? 'User'}>
      {user && (
        <dl className="grid grid-cols-[120px_1fr] gap-y-2">
          <dt className="text-muted">Email</dt><dd>{user.email}{user.emailVerified ? ' ✓' : ''}</dd>
          <dt className="text-muted">Sign-in</dt><dd>{[user.hasPassword ? 'password' : null, ...user.providers].filter(Boolean).join(', ') || '—'}</dd>
          <dt className="text-muted">Locale</dt><dd>{user.preferences.currency} · {user.preferences.timezone} · {user.preferences.theme}</dd>
          <dt className="text-muted">Joined</dt><dd>{formatDate(user.createdAt)}</dd>
          <dt className="text-muted">Receipts</dt><dd>{q.data ? q.data.receiptCount : '…'}</dd>
          <dt className="text-muted">Status</dt><dd>{user.disabled ? 'Disabled' : 'Active'}</dd>
          <dt className="text-muted">ID</dt><dd className="font-mono text-xs">{user.id}</dd>
        </dl>
      )}
    </Modal>
  )
}
