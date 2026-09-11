import { useState, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Search, UserCog, UserPlus, Ban, CircleCheck, KeyRound, RefreshCw } from 'lucide-react'
import { clsx } from 'clsx'
import { PageHeader } from '@common/ui/PageHeader'
import { Field, Input, Select } from '@common/ui/Field'
import { Button } from '@common/ui/Button'
import { Skeleton } from '@common/ui/Skeleton'
import { EmptyState } from '@common/ui/EmptyState'
import { Modal } from '@common/ui/Modal'
import { Alert } from '@common/ui/Alert'
import { ApiRequestError } from '@common/api/client'
import { useDebounce } from '@common/hooks/useDebounce'
import { useAuth } from '@/features/auth/useAuth'
import type { Admin } from '@/features/auth/types'
import { adminsApi, adminKeys, generatePassword, type AdminRole } from './api'

const PAGE = 20
const errMsg = (e: unknown) => (e instanceof ApiRequestError ? e.message : 'Something went wrong')
const fieldErr = (e: unknown, f: string) => (e instanceof ApiRequestError ? e.fields?.[f] : undefined)
const lastLogin = (a: Admin) => (a.lastLoginAt ? new Date(a.lastLoginAt).toLocaleString('en-IN', { dateStyle: 'medium', timeStyle: 'short' }) : 'Never')

export function AdminsPage() {
  const qc = useQueryClient()
  const { user: me } = useAuth()
  const isSuper = me?.role === 'superadmin'
  const [q, setQ] = useState('')
  const [page, setPage] = useState(1)
  const [adding, setAdding] = useState(false)
  const [pwTarget, setPwTarget] = useState<Admin | null>(null)
  const [error, setError] = useState('')
  const dq = useDebounce(q, 300)
  const params = { q: dq, page, pageSize: PAGE }
  const list = useQuery({ queryKey: adminKeys.list(params), queryFn: () => adminsApi.list(params), placeholderData: (prev) => prev })
  const pages = Math.max(1, Math.ceil((list.data?.total ?? 0) / PAGE))
  const refresh = () => qc.invalidateQueries({ queryKey: adminKeys.all })

  const toggle = useMutation({
    mutationFn: ({ id, disable }: { id: string; disable: boolean }) => (disable ? adminsApi.disable(id) : adminsApi.enable(id)),
    onSuccess: () => { setError(''); refresh() },
    onError: (e) => setError(errMsg(e)),
  })
  const role = useMutation({
    mutationFn: ({ id, role }: { id: string; role: AdminRole }) => adminsApi.update(id, { role }),
    onSuccess: () => { setError(''); refresh() },
    onError: (e) => { setError(errMsg(e)); refresh() },
  })

  return (
    <div>
      <PageHeader
        title="Admins"
        subtitle={list.data ? `${list.data.total.toLocaleString('en-IN')} admin accounts` : undefined}
        actions={isSuper ? (
          <Button size="sm" onClick={() => setAdding(true)}>
            <UserPlus className="size-4" /> Add admin
          </Button>
        ) : undefined}
      />
      {!isSuper && <div className="mb-4"><Alert kind="info">Only superadmins can add, disable or change admin accounts. This list is read-only for you.</Alert></div>}
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="relative w-full max-w-md">
          <Search className="text-muted pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2" />
          <Input
            placeholder="Search by email or name"
            value={q}
            onChange={(e) => { setQ(e.target.value); setPage(1) }}
            className="pl-9"
            aria-label="Search admins"
          />
        </div>
      </div>
      {error && <div className="mb-4"><Alert>{error}</Alert></div>}
      {list.error && <Alert>Could not load admins.</Alert>}
      <div className="card overflow-x-auto">
        {list.isLoading ? (
          <div className="grid gap-2 p-4">{Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-10" />)}</div>
        ) : !list.data?.admins.length ? (
          <EmptyState icon={UserCog} title="No admins found" description={dq ? 'Try a different search.' : 'Create the first admin with the create-admin script.'} />
        ) : (
          <table className="w-full min-w-[720px] text-sm">
            <thead className="text-muted text-left text-xs uppercase">
              <tr className="border-b border-[var(--border)]">
                <th className="px-4 py-3 font-medium">Admin</th>
                <th className="px-4 py-3 font-medium">Role</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">Last login</th>
                {isSuper && <th className="px-4 py-3 text-right font-medium">Actions</th>}
              </tr>
            </thead>
            <tbody className="divide-y divide-[var(--border)]">
              {list.data.admins.map((a) => {
                const self = a.id === me?.id
                return (
                  <tr key={a.id} data-email={a.email}>
                    <td className="px-4 py-3">
                      <div className="font-medium">{a.name}{self && <span className="text-muted ml-1 text-xs">(you)</span>}</div>
                      <div className="text-muted text-xs">{a.email}</div>
                    </td>
                    <td className="px-4 py-3">
                      {isSuper && !self ? (
                        <Select
                          aria-label={`Role for ${a.email}`}
                          value={a.role}
                          disabled={role.isPending}
                          onChange={(e) => role.mutate({ id: a.id, role: e.target.value as AdminRole })}
                          className="h-8 w-auto! text-xs"
                        >
                          <option value="admin">admin</option>
                          <option value="superadmin">superadmin</option>
                        </Select>
                      ) : (
                        <RoleBadge role={a.role} />
                      )}
                    </td>
                    <td className="px-4 py-3"><StatusBadge disabled={!!a.disabled} /></td>
                    <td className="px-4 py-3 whitespace-nowrap">{lastLogin(a)}</td>
                    {isSuper && (
                      <td className="px-4 py-3">
                        <div className="flex justify-end gap-2">
                          {!self && (a.disabled ? (
                            <Button variant="secondary" size="sm" aria-label={`Enable ${a.email}`} loading={toggle.isPending && toggle.variables?.id === a.id} onClick={() => toggle.mutate({ id: a.id, disable: false })}>
                              <CircleCheck className="size-4" /> Enable
                            </Button>
                          ) : (
                            <Button variant="secondary" size="sm" aria-label={`Disable ${a.email}`} loading={toggle.isPending && toggle.variables?.id === a.id} onClick={() => toggle.mutate({ id: a.id, disable: true })}>
                              <Ban className="size-4" /> Disable
                            </Button>
                          ))}
                          <Button variant="secondary" size="sm" aria-label={`Set password for ${a.email}`} onClick={() => setPwTarget(a)}>
                            <KeyRound className="size-4" /> Set password
                          </Button>
                        </div>
                      </td>
                    )}
                  </tr>
                )
              })}
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
      <AddAdminModal open={adding} onClose={() => setAdding(false)} onDone={refresh} />
      <SetPasswordModal key={pwTarget?.id ?? 'none'} admin={pwTarget} onClose={() => setPwTarget(null)} onDone={refresh} />
    </div>
  )
}

function RoleBadge({ role }: { role: AdminRole }) {
  return (
    <span className={clsx('inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
      role === 'superadmin' ? 'bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300' : 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300')}>
      {role}
    </span>
  )
}

function StatusBadge({ disabled }: { disabled: boolean }) {
  return (
    <span className={clsx('inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
      disabled ? 'bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300' : 'bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300')}>
      {disabled ? 'Disabled' : 'Active'}
    </span>
  )
}

function PasswordField({ value, onChange, error }: { value: string; onChange: (v: string) => void; error?: string }) {
  return (
    <Field label="Password" error={error} hint="At least 8 characters. Generate one or type your own; share it with the admin directly.">
      <div className="flex gap-2">
        <Input name="password" type="text" autoComplete="off" spellCheck={false} value={value} onChange={(e) => onChange(e.target.value)} className="font-mono" invalid={!!error} />
        <Button type="button" variant="secondary" className="h-10 shrink-0" onClick={() => onChange(generatePassword())} aria-label="Generate password">
          <RefreshCw className="size-4" /> Generate
        </Button>
      </div>
    </Field>
  )
}

function AddAdminModal({ open, onClose, onDone }: { open: boolean; onClose: () => void; onDone: () => void }) {
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [role, setRole] = useState<AdminRole>('admin')
  const [password, setPassword] = useState('')
  const create = useMutation({
    mutationFn: () => adminsApi.create({ name: name.trim(), email: email.trim(), password, role }),
    onSuccess: () => { onDone(); onClose(); setName(''); setEmail(''); setRole('admin'); setPassword('') },
  })
  const submit = (e: FormEvent) => { e.preventDefault(); create.mutate() }
  const err = create.error
  return (
    <Modal open={open} onClose={onClose} title="Add admin">
      <form onSubmit={submit} className="grid gap-3">
        {err && !fieldErr(err, 'email') && !fieldErr(err, 'password') && !fieldErr(err, 'name') && <Alert>{errMsg(err)}</Alert>}
        <Field label="Name" error={fieldErr(err, 'name')}>
          <Input name="name" value={name} onChange={(e) => setName(e.target.value)} required maxLength={80} invalid={!!fieldErr(err, 'name')} />
        </Field>
        <Field label="Email" error={fieldErr(err, 'email')}>
          <Input name="email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required invalid={!!fieldErr(err, 'email')} />
        </Field>
        <Field label="Role">
          <Select name="role" value={role} onChange={(e) => setRole(e.target.value as AdminRole)}>
            <option value="admin">admin — manage users, categories, config</option>
            <option value="superadmin">superadmin — also delete users and manage admins</option>
          </Select>
        </Field>
        <PasswordField value={password} onChange={(v) => { setPassword(v); if (err) create.reset() }} error={fieldErr(err, 'password')} />
        <div className="mt-2 flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>Cancel</Button>
          <Button type="submit" loading={create.isPending}>Create admin</Button>
        </div>
      </form>
    </Modal>
  )
}

function SetPasswordModal({ admin, onClose, onDone }: { admin: Admin | null; onClose: () => void; onDone: () => void }) {
  const [password, setPassword] = useState('')
  const [done, setDone] = useState(false)
  const set = useMutation({
    mutationFn: () => adminsApi.resetPassword(admin!.id, password),
    onSuccess: () => { onDone(); setDone(true) },
  })
  const submit = (e: FormEvent) => { e.preventDefault(); set.mutate() }
  const err = set.error
  return (
    <Modal open={!!admin} onClose={onClose} title={`Set password for ${admin?.name ?? ''}`}>
      {done ? (
        <div className="grid gap-3">
          <Alert kind="success">Password updated for {admin?.email}. Their other sessions were signed out.</Alert>
          <div className="flex justify-end"><Button onClick={onClose}>Done</Button></div>
        </div>
      ) : (
        <form onSubmit={submit} className="grid gap-3">
          <p className="text-muted">{admin?.email}</p>
          {err && !fieldErr(err, 'password') && <Alert>{errMsg(err)}</Alert>}
          <PasswordField value={password} onChange={(v) => { setPassword(v); if (err) set.reset() }} error={fieldErr(err, 'password')} />
          <div className="mt-2 flex justify-end gap-2">
            <Button type="button" variant="secondary" onClick={onClose}>Cancel</Button>
            <Button type="submit" loading={set.isPending}>Set password</Button>
          </div>
        </form>
      )}
    </Modal>
  )
}
