import { useState } from 'react'
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router'
import { Bell, BellOff, CheckCheck, ChevronLeft, ChevronRight, CircleCheck, PiggyBank, ShieldAlert, TriangleAlert, BarChart3, type LucideIcon } from 'lucide-react'
import { clsx } from 'clsx'
import { Button } from '@common/ui/Button'
import { EmptyState } from '@common/ui/EmptyState'
import { Skeleton } from '@common/ui/Skeleton'
import { notificationsApi, notificationKeys, notificationLink, type Notification, type NotificationType } from './api'

const icons: Record<NotificationType, [LucideIcon, string]> = {
  budget_alert: [PiggyBank, 'bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300'],
  warranty_expiring: [ShieldAlert, 'bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300'],
  extraction_done: [CircleCheck, 'bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300'],
  extraction_failed: [TriangleAlert, 'bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300'],
  weekly_summary: [BarChart3, 'bg-blue-50 text-blue-700 dark:bg-blue-950/40 dark:text-blue-300'],
}

function relative(iso: string): string {
  const diff = (Date.now() - new Date(iso).getTime()) / 1000
  if (diff < 60) return 'just now'
  if (diff < 3600) return `${Math.floor(diff / 60)} min ago`
  if (diff < 86400) return `${Math.floor(diff / 3600)} h ago`
  if (diff < 7 * 86400) return `${Math.floor(diff / 86400)} d ago`
  return new Intl.DateTimeFormat('en-IN', { day: '2-digit', month: 'short' }).format(new Date(iso))
}

export function NotificationsPage() {
  const qc = useQueryClient()
  const nav = useNavigate()
  const [page, setPage] = useState(1)
  const q = useQuery({ queryKey: notificationKeys.list(page), queryFn: () => notificationsApi.list(page, 20), placeholderData: keepPreviousData })
  const invalidate = () => qc.invalidateQueries({ queryKey: notificationKeys.all })
  const readOne = useMutation({ mutationFn: (id: string) => notificationsApi.read(id), onSuccess: invalidate })
  const readAll = useMutation({ mutationFn: notificationsApi.readAll, onSuccess: invalidate })

  const open = (n: Notification) => {
    if (!n.readAt) readOne.mutate(n.id)
    const to = notificationLink(n)
    if (to) nav(to)
  }
  const rows = q.data?.notifications ?? []
  const pages = Math.max(1, Math.ceil((q.data?.total ?? 0) / 20))

  return (
    <div className="mx-auto max-w-3xl">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Notifications</h1>
          <p className="text-muted text-sm">{q.data ? (q.data.unread ? `${q.data.unread} unread` : 'All caught up') : ''}</p>
        </div>
        <Button size="sm" variant="secondary" onClick={() => readAll.mutate()} disabled={!q.data?.unread} loading={readAll.isPending}><CheckCheck className="size-4" /> Mark all read</Button>
      </div>

      {q.isLoading ? (
        <div className="grid gap-2"><Skeleton className="h-16" /><Skeleton className="h-16" /><Skeleton className="h-16" /></div>
      ) : rows.length === 0 ? (
        <div className="card"><EmptyState icon={BellOff} title="No notifications yet" description="Budget alerts, warranty reminders and extraction updates will show up here." /></div>
      ) : (
        <ul className="card divide-y divide-[var(--border)] overflow-hidden">
          {rows.map((n) => {
            const [Icon, cls] = icons[n.type] ?? [Bell, '']
            const unread = !n.readAt
            return (
              <li key={n.id}>
                <button type="button" onClick={() => open(n)} className={clsx('flex w-full items-start gap-3 px-4 py-3 text-left hover:bg-brand-50/50 dark:hover:bg-brand-900/10', unread && 'bg-brand-50/30 dark:bg-brand-900/10')} data-unread={unread || undefined}>
                  <span className={clsx('mt-0.5 grid size-9 shrink-0 place-items-center rounded-lg', cls)}><Icon className="size-4.5" /></span>
                  <span className="min-w-0 flex-1">
                    <span className="flex items-center gap-2">
                      <span className={clsx('truncate text-sm', unread ? 'font-semibold' : 'font-medium')}>{n.title}</span>
                      {unread && <span className="size-2 shrink-0 rounded-full bg-brand-600" aria-label="Unread" />}
                    </span>
                    <span className="text-muted mt-0.5 block text-sm">{n.body}</span>
                    <span className="text-muted mt-1 block text-xs">{relative(n.createdAt)}</span>
                  </span>
                </button>
              </li>
            )
          })}
        </ul>
      )}

      {pages > 1 && (
        <div className="mt-4 flex items-center justify-center gap-3 text-sm">
          <Button size="sm" variant="secondary" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}><ChevronLeft className="size-4" /> Prev</Button>
          <span className="text-muted">Page {page} of {pages}</span>
          <Button size="sm" variant="secondary" disabled={page >= pages} onClick={() => setPage((p) => p + 1)}>Next <ChevronRight className="size-4" /></Button>
        </div>
      )}
    </div>
  )
}
