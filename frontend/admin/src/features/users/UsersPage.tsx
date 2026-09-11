import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Search, Users, Ban, CircleCheck, KeyRound, Trash2, Copy, Check } from 'lucide-react'
import { clsx } from 'clsx'
import { PageHeader } from '@common/ui/PageHeader'
import { Input } from '@common/ui/Field'
import { Button } from '@common/ui/Button'
import { Chip } from '@common/ui/Chip'
import { Skeleton } from '@common/ui/Skeleton'
import { EmptyState } from '@common/ui/EmptyState'
import { Modal } from '@common/ui/Modal'
import { Alert } from '@common/ui/Alert'
import { ApiRequestError } from '@common/api/client'
import { useDebounce } from '@common/hooks/useDebounce'
import { formatDate } from '@common/format/date'
import { useAuth } from '@/features/auth/useAuth'
import { usersApi, userKeys, type PortalUser, type UserStatusFilter } from './api'

const PAGE = 20
const statusFilters: { value: UserStatusFilter; label: string }[] = [
  { value: '', label: 'All' },
  { value: 'active', label: 'Active' },
  { value: 'disabled', label: 'Disabled' },
]

const signIn = (u: PortalUser) => [u.hasPassword ? 'password' : null, ...u.providers].filter(Boolean).join(', ') || '—'
const errMsg = (e: unknown) => (e instanceof ApiRequestError ? e.message : 'Something went wrong')

export function UsersPage() {
  const [q, setQ] = useState('')
  const [status, setStatus] = useState<UserStatusFilter>('')
  const [page, setPage] = useState(1)
  const [selected, setSelected] = useState<PortalUser | null>(null)
  const dq = useDebounce(q, 300)
  const params = { q: dq, status, page, pageSize: PAGE }
  const list = useQuery({ queryKey: userKeys.list(params), queryFn: () => usersApi.list(params), placeholderData: (prev) => prev })
  const pages = Math.max(1, Math.ceil((list.data?.total ?? 0) / PAGE))

  return (
    <div>
      <PageHeader title="Users" subtitle={list.data ? `${list.data.total.toLocaleString('en-IN')} accounts` : undefined} />
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="relative w-full max-w-md">
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
        <div className="flex gap-2" role="group" aria-label="Filter by status">
          {statusFilters.map((f) => (
            <Chip key={f.label} active={status === f.value} aria-pressed={status === f.value} onClick={() => { setStatus(f.value); setPage(1) }}>
              {f.label}
            </Chip>
          ))}
        </div>
      </div>
      {list.error && <Alert>Could not load users.</Alert>}
      <div className="card overflow-x-auto">
        {list.isLoading ? (
          <div className="grid gap-2 p-4">{Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-10" />)}</div>
        ) : !list.data?.users.length ? (
          <EmptyState icon={Users} title="No users found" description={dq || status ? 'Try a different search or filter.' : 'Nobody has signed up yet.'} />
        ) : (
          <table className="w-full min-w-[640px] text-sm">
            <thead className="text-muted text-left text-xs uppercase">
              <tr className="border-b border-[var(--border)]">
                <th className="px-4 py-3 font-medium">User</th>
                <th className="px-4 py-3 font-medium">Sign-in</th>
                <th className="px-4 py-3 font-medium">Status</th>
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
                  <td className="px-4 py-3">{signIn(u)}</td>
                  <td className="px-4 py-3"><UserStatus disabled={!!u.disabled} /></td>
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
      <UserDetail key={selected?.id ?? 'none'} user={selected} onClose={() => setSelected(null)} />
    </div>
  )
}

function UserStatus({ disabled }: { disabled: boolean }) {
  return (
    <span className={clsx('inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
      disabled ? 'bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300' : 'bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300')}>
      {disabled ? 'Disabled' : 'Active'}
    </span>
  )
}

function UserDetail({ user, onClose }: { user: PortalUser | null; onClose: () => void }) {
  const qc = useQueryClient()
  const { user: admin } = useAuth()
  const isSuper = admin?.role === 'superadmin'
  const q = useQuery({ queryKey: userKeys.detail(user?.id ?? ''), queryFn: () => usersApi.get(user!.id), enabled: !!user })
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [reset, setReset] = useState<{ link: string; expiresAt: string } | null>(null)
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState('')
  const u = q.data?.user ?? user

  const toggle = useMutation({
    mutationFn: (disable: boolean) => (disable ? usersApi.disable(user!.id) : usersApi.enable(user!.id)),
    onSuccess: () => { setError(''); qc.invalidateQueries({ queryKey: userKeys.all }) },
    onError: (e) => setError(errMsg(e)),
  })
  const link = useMutation({
    mutationFn: () => usersApi.resetLink(user!.id),
    onSuccess: (r) => { setError(''); setCopied(false); setReset(r) },
    onError: (e) => setError(errMsg(e)),
  })
  const remove = useMutation({
    mutationFn: () => usersApi.remove(user!.id),
    onSuccess: () => {
      const id = user!.id
      onClose()
      qc.removeQueries({ queryKey: userKeys.detail(id) })
      qc.invalidateQueries({ queryKey: userKeys.all })
    },
    onError: (e) => setError(errMsg(e)),
  })
  const copy = async () => {
    if (!reset) return
    try {
      await navigator.clipboard.writeText(reset.link)
      setCopied(true)
    } catch {
      setError('Copy failed. Select the link and copy it manually.')
    }
  }

  return (
    <Modal open={!!user} onClose={onClose} title={u?.name ?? 'User'}>
      {u && (
        <div className="grid gap-4">
          {error && <Alert>{error}</Alert>}
          <dl className="grid grid-cols-[96px_minmax(0,1fr)] gap-y-2">
            <dt className="text-muted">Email</dt><dd className="break-all">{u.email}{u.emailVerified ? ' ✓' : ''}</dd>
            <dt className="text-muted">Sign-in</dt><dd>{signIn(u)}</dd>
            <dt className="text-muted">Locale</dt><dd>{u.preferences.currency} · {u.preferences.timezone}</dd>
            <dt className="text-muted">Joined</dt><dd>{formatDate(u.createdAt)}</dd>
            <dt className="text-muted">Receipts</dt><dd>{q.data ? q.data.receiptCount : '…'}</dd>
            <dt className="text-muted">Status</dt><dd><UserStatus disabled={!!u.disabled} /></dd>
            <dt className="text-muted">ID</dt><dd className="font-mono text-xs break-all">{u.id}</dd>
          </dl>

          {reset && (
            <div className="rounded-lg border border-[var(--border)] p-3">
              <div className="mb-1 text-xs font-medium">Password reset link</div>
              <div className="flex gap-2">
                <Input readOnly value={reset.link} aria-label="Reset link" className="font-mono text-xs" onFocus={(e) => e.currentTarget.select()} />
                <Button variant="secondary" size="sm" className="h-10 shrink-0" onClick={copy} aria-label="Copy reset link">
                  {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
                </Button>
              </div>
              <p className="text-muted mt-1 text-xs">
                Single use, expires {new Date(reset.expiresAt).toLocaleTimeString('en-IN', { hour: '2-digit', minute: '2-digit' })}. Share it with the user directly; nothing is emailed.
              </p>
            </div>
          )}

          {confirmDelete ? (
            <div className="rounded-lg border border-red-200 bg-red-50 p-3 dark:border-red-900 dark:bg-red-950/30">
              <p className="text-sm">
                Delete <strong>{u.email}</strong> and all of their receipts, files, budgets, warranties and notifications? This cannot be undone.
              </p>
              <div className="mt-3 flex justify-end gap-2">
                <Button variant="secondary" size="sm" onClick={() => setConfirmDelete(false)}>Cancel</Button>
                <Button variant="danger" size="sm" loading={remove.isPending} onClick={() => remove.mutate()}>Delete account</Button>
              </div>
            </div>
          ) : (
            <div className="flex flex-wrap gap-2 border-t border-[var(--border)] pt-4">
              {u.disabled ? (
                <Button variant="secondary" size="sm" loading={toggle.isPending} onClick={() => toggle.mutate(false)}>
                  <CircleCheck className="size-4" /> Enable
                </Button>
              ) : (
                <Button variant="secondary" size="sm" loading={toggle.isPending} onClick={() => toggle.mutate(true)}>
                  <Ban className="size-4" /> Disable
                </Button>
              )}
              <Button variant="secondary" size="sm" loading={link.isPending} onClick={() => link.mutate()}>
                <KeyRound className="size-4" /> Reset link
              </Button>
              <Button
                variant="danger"
                size="sm"
                className="sm:ml-auto"
                disabled={!isSuper}
                title={isSuper ? undefined : 'Only superadmins can delete accounts'}
                onClick={() => setConfirmDelete(true)}
              >
                <Trash2 className="size-4" /> Delete
              </Button>
            </div>
          )}
          {!isSuper && !confirmDelete && <p className="text-muted -mt-2 text-xs">Only superadmins can delete accounts.</p>}
        </div>
      )}
    </Modal>
  )
}
