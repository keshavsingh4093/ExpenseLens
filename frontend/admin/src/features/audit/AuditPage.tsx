import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ScrollText, X } from 'lucide-react'
import { clsx } from 'clsx'
import { PageHeader } from '@common/ui/PageHeader'
import { Field, Input, Select } from '@common/ui/Field'
import { Button } from '@common/ui/Button'
import { Skeleton } from '@common/ui/Skeleton'
import { EmptyState } from '@common/ui/EmptyState'
import { Alert } from '@common/ui/Alert'
import { useDebounce } from '@common/hooks/useDebounce'
import { auditApi, auditKeys, type AuditEntry } from './api'

const PAGE = 25
const destructive = new Set(['user.delete', 'user.disable', 'category.delete', 'system_config.delete'])

const when = (iso: string) =>
  new Date(iso).toLocaleString('en-IN', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit', second: '2-digit' })

function detailText(d?: Record<string, unknown>) {
  if (!d) return ''
  return Object.entries(d)
    .map(([k, v]) => `${k}: ${Array.isArray(v) ? v.join(', ') : typeof v === 'object' && v !== null ? JSON.stringify(v) : String(v)}`)
    .join(' · ')
}

export function AuditPage() {
  const [action, setAction] = useState('')
  const [admin, setAdmin] = useState<{ id: string; email: string } | null>(null)
  const [target, setTarget] = useState('')
  const [page, setPage] = useState(1)
  const dTarget = useDebounce(target.trim(), 300)
  const params = { action, adminId: admin?.id, targetId: dTarget, page, pageSize: PAGE }
  const list = useQuery({ queryKey: auditKeys.list(params), queryFn: () => auditApi.list(params), placeholderData: (prev) => prev })
  const actions = useQuery({ queryKey: auditKeys.actions, queryFn: auditApi.actions })
  const pages = Math.max(1, Math.ceil((list.data?.total ?? 0) / PAGE))
  const filtered = !!(action || admin || dTarget)

  return (
    <div>
      <PageHeader title="Audit log" subtitle={list.data ? `${list.data.total.toLocaleString('en-IN')} admin actions` : 'Every change made from the admin portal'} />
      <div className="mb-4 grid gap-3 sm:grid-cols-[220px_minmax(0,280px)_auto] sm:items-end">
        <Field label="Action">
          <Select value={action} onChange={(e) => { setAction(e.target.value); setPage(1) }} aria-label="Filter by action">
            <option value="">All actions</option>
            {(actions.data?.actions ?? []).slice().sort().map((a) => <option key={a} value={a}>{a}</option>)}
          </Select>
        </Field>
        <Field label="Target ID">
          <Input value={target} onChange={(e) => { setTarget(e.target.value); setPage(1) }} placeholder="user, receipt or category id" className="font-mono" aria-label="Filter by target id" />
        </Field>
        {filtered && (
          <Button variant="ghost" size="sm" className="h-10 justify-self-start" onClick={() => { setAction(''); setAdmin(null); setTarget(''); setPage(1) }}>
            Clear filters
          </Button>
        )}
      </div>
      {admin && (
        <div className="mb-3 flex items-center gap-2 text-sm">
          <span className="text-muted">Admin:</span>
          <span className="inline-flex items-center gap-1 rounded-full bg-brand-50 px-2.5 py-1 text-xs font-medium text-brand-700 dark:bg-brand-900/30 dark:text-brand-300">
            {admin.email}
            <button type="button" aria-label="Clear admin filter" onClick={() => { setAdmin(null); setPage(1) }}><X className="size-3.5" /></button>
          </span>
        </div>
      )}
      {list.error && <Alert>Could not load the audit log.</Alert>}
      <div className="card overflow-x-auto">
        {list.isLoading ? (
          <div className="grid gap-2 p-4">{Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} className="h-10" />)}</div>
        ) : !list.data?.entries.length ? (
          <EmptyState icon={ScrollText} title="No entries" description={filtered ? 'Nothing matches these filters.' : 'Admin actions appear here as they happen.'} />
        ) : (
          <table className="w-full min-w-[760px] text-sm">
            <thead className="text-muted text-left text-xs uppercase">
              <tr className="border-b border-[var(--border)]">
                <th className="px-4 py-3 font-medium">When</th>
                <th className="px-4 py-3 font-medium">Admin</th>
                <th className="px-4 py-3 font-medium">Action</th>
                <th className="px-4 py-3 font-medium">Target</th>
                <th className="px-4 py-3 font-medium">Details</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[var(--border)]">
              {list.data.entries.map((e) => <Row key={e.id} e={e} onAdmin={() => { setAdmin({ id: e.adminId, email: e.adminEmail }); setPage(1) }} onTarget={() => { setTarget(e.targetId); setPage(1) }} />)}
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
    </div>
  )
}

function Row({ e, onAdmin, onTarget }: { e: AuditEntry; onAdmin: () => void; onTarget: () => void }) {
  return (
    <tr className="align-top">
      <td className="px-4 py-3 whitespace-nowrap tabular-nums">{when(e.at)}</td>
      <td className="px-4 py-3">
        <button type="button" className="text-left hover:text-brand-700 hover:underline" onClick={onAdmin} title="Show only this admin">{e.adminEmail}</button>
        {e.ip && <div className="text-muted font-mono text-xs">{e.ip}</div>}
      </td>
      <td className="px-4 py-3">
        <span className={clsx('rounded-md px-1.5 py-0.5 font-mono text-xs whitespace-nowrap',
          destructive.has(e.action) ? 'bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300' : 'bg-gray-100 dark:bg-gray-800')}>
          {e.action}
        </span>
      </td>
      <td className="px-4 py-3">
        <div className="text-muted text-xs">{e.targetType.replace('_', ' ')}</div>
        {e.targetId && (
          <button type="button" className="font-mono text-xs break-all hover:text-brand-700 hover:underline" onClick={onTarget} title="Show only this target">
            {e.targetId}
          </button>
        )}
      </td>
      <td className="text-muted max-w-[280px] px-4 py-3 text-xs break-words">{detailText(e.details) || '—'}</td>
    </tr>
  )
}
