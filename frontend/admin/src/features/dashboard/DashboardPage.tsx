import { useQuery } from '@tanstack/react-query'
import { Users, UserPlus, Receipt, AlertTriangle } from 'lucide-react'
import { PageHeader } from '@common/ui/PageHeader'
import { Skeleton } from '@common/ui/Skeleton'
import { Alert } from '@common/ui/Alert'
import { StatTile } from '@common/charts/StatTile'
import { StatusBadge } from '@common/ui/StatusBadge'
import type { ReceiptStatus } from '@common/types'
import { statsApi, statsKeys } from './api'

const statuses: ReceiptStatus[] = ['processing', 'review', 'confirmed', 'failed']

export function DashboardPage() {
  const q = useQuery({ queryKey: statsKeys.overview, queryFn: statsApi.overview, refetchInterval: 30_000 })
  const o = q.data
  return (
    <div>
      <PageHeader title="Dashboard" subtitle="Platform overview" />
      {q.error && <Alert>Could not load stats.</Alert>}
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {!o ? (
          Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-28" />)
        ) : (
          <>
            <StatTile label="Total users" value={o.users.total.toLocaleString('en-IN')} icon={Users} />
            <StatTile label="New users (7 days)" value={o.users.last7Days.toLocaleString('en-IN')} icon={UserPlus} />
            <StatTile label="Receipts" value={o.receipts.total.toLocaleString('en-IN')} icon={Receipt} />
            <StatTile label="Extraction failures" value={`${o.receipts.failed} (${o.receipts.failurePct.toFixed(1)}%)`} icon={AlertTriangle} />
          </>
        )}
      </div>
      {o && (
        <div className="mt-4 grid gap-4 lg:grid-cols-2">
          <section className="card p-4">
            <h2 className="text-base font-semibold">Receipts by status</h2>
            <ul className="mt-3 divide-y divide-[var(--border)]">
              {statuses.map((s) => (
                <li key={s} className="flex items-center justify-between py-2 text-sm">
                  <StatusBadge status={s} />
                  <span className="font-medium tabular-nums">{(o.receipts.byStatus[s] ?? 0).toLocaleString('en-IN')}</span>
                </li>
              ))}
            </ul>
          </section>
          <section className="card p-4">
            <h2 className="text-base font-semibold">Activity</h2>
            <dl className="mt-3 grid grid-cols-2 gap-3 text-sm">
              <Row k="Users with receipts" v={o.users.withReceipts} />
              <Row k="New users (30 days)" v={o.users.last30Days} />
              <Row k="Receipts (7 days)" v={o.receipts.last7Days} />
              <Row k="Receipts (30 days)" v={o.receipts.last30Days} />
            </dl>
            <p className="text-muted mt-3 text-xs">Updated {new Date(o.generatedAt).toLocaleTimeString('en-IN')}</p>
          </section>
        </div>
      )}
    </div>
  )
}

function Row({ k, v }: { k: string; v: number }) {
  return (
    <div className="rounded-lg border border-[var(--border)] p-3">
      <dt className="text-muted text-xs">{k}</dt>
      <dd className="mt-1 text-lg font-semibold tabular-nums">{v.toLocaleString('en-IN')}</dd>
    </div>
  )
}
