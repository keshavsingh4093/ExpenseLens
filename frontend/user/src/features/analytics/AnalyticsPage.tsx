import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Wallet, Hash, Gauge, Flame, Receipt as ReceiptIcon, Store } from 'lucide-react'
import { Chip } from '@common/ui/Chip'
import { Skeleton } from '@common/ui/Skeleton'
import { EmptyState } from '@common/ui/EmptyState'
import { TrendChart } from '@common/charts/TrendChart'
import { CategoryDonut } from '@common/charts/CategoryDonut'
import { toSlices } from '@common/charts/donutSlices'
import type { RangeName } from '@common/analytics/types'
import { formatMoney } from '@common/format/money'
import { formatDate } from '@common/format/date'
import { useCategories } from '@/features/categories/useCategories'
import { useAuth } from '@/features/auth/useAuth'
import { analyticsApi, analyticsKeys } from './api'

const ranges: [RangeName, string][] = [['7d', '7 days'], ['30d', '30 days'], ['3m', '3 months'], ['6m', '6 months'], ['1y', '1 year'], ['all', 'All time']]

export function AnalyticsPage() {
  const { user } = useAuth()
  const currency = user?.preferences.currency ?? 'INR'
  const { byId } = useCategories()
  const [range, setRange] = useState<RangeName>('30d')
  const p = { range }
  const trend = useQuery({ queryKey: analyticsKeys.trend(p), queryFn: () => analyticsApi.trend(p) })
  const cats = useQuery({ queryKey: analyticsKeys.categories(p), queryFn: () => analyticsApi.categories(p) })
  const merchants = useQuery({ queryKey: analyticsKeys.merchants({ ...p, limit: 6 }), queryFn: () => analyticsApi.merchants({ ...p, limit: 6 }) })
  const stats = useQuery({ queryKey: analyticsKeys.stats(p), queryFn: () => analyticsApi.stats(p) })

  const s = stats.data?.stats
  const slices = cats.data ? toSlices(cats.data.categories, byId) : []
  const catTotal = cats.data?.categories.reduce((a, c) => a + c.total, 0) ?? 0
  const hasTrend = trend.data?.points.some((pt) => pt.total > 0)
  const topMax = merchants.data?.merchants[0]?.total ?? 0

  return (
    <div className="mx-auto max-w-6xl">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Analytics</h1>
          <p className="text-muted text-sm">Where your money goes.</p>
        </div>
        <div className="flex gap-2 overflow-x-auto pb-1" role="tablist" aria-label="Range">
          {ranges.map(([v, l]) => <Chip key={v} role="tab" aria-selected={range === v} active={range === v} onClick={() => setRange(v)}>{l}</Chip>)}
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        {stats.isLoading || !s ? (
          Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-24" />)
        ) : (
          <>
            <Stat icon={Wallet} label="Total spent" value={formatMoney(s.total, currency)} sub={`${s.days} day${s.days === 1 ? '' : 's'}`} />
            <Stat icon={Hash} label="Transactions" value={String(s.transactions)} sub={`${formatMoney(s.avgPerTransaction, currency)} per receipt`} />
            <Stat icon={Gauge} label="Average daily" value={formatMoney(s.avgDaily, currency)} />
            <Stat icon={Flame} label="Highest day" value={s.highestDay ? formatMoney(s.highestDay.total, currency) : '—'} sub={s.highestDay ? formatDate(s.highestDay.date) : 'No spending yet'} />
          </>
        )}
      </div>

      <div className="mt-4 grid gap-4 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
        <section className="card p-4 lg:col-span-2">
          <h2 className="mb-2 text-sm font-semibold">Spending trend</h2>
          {trend.isLoading ? <Skeleton className="h-[240px]" /> : hasTrend ? (
            <TrendChart points={trend.data!.points} currency={currency} height={240} />
          ) : (
            <EmptyState icon={ReceiptIcon} title="No spending in this period" description="Upload receipts to see your trend." />
          )}
        </section>

        <section className="card p-4">
          <h2 className="mb-3 text-sm font-semibold">Category breakdown</h2>
          {cats.isLoading ? <Skeleton className="h-[180px]" /> : slices.length ? (
            <CategoryDonut slices={slices} total={catTotal} currency={currency} />
          ) : (
            <EmptyState icon={ReceiptIcon} title="Nothing categorized yet" />
          )}
        </section>

        <section className="card p-4">
          <h2 className="mb-3 text-sm font-semibold">Top merchants</h2>
          {merchants.isLoading ? <Skeleton className="h-[180px]" /> : merchants.data?.merchants.length ? (
            <ul className="grid gap-3">
              {merchants.data.merchants.map((m) => (
                <li key={m.merchant}>
                  <div className="flex items-center justify-between gap-2 text-sm">
                    <span className="flex min-w-0 items-center gap-2"><Store className="text-muted size-4 shrink-0" /><span className="truncate font-medium">{m.merchant}</span><span className="text-muted shrink-0 text-xs">· {m.count}</span></span>
                    <span className="shrink-0 font-semibold tabular-nums">{formatMoney(m.total, currency)}</span>
                  </div>
                  <div className="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                    <div className="h-full rounded-full bg-brand-600" style={{ width: `${topMax ? (m.total * 100) / topMax : 0}%` }} />
                  </div>
                </li>
              ))}
            </ul>
          ) : (
            <EmptyState icon={Store} title="No merchants yet" />
          )}
        </section>
      </div>
    </div>
  )
}

function Stat({ icon: Icon, label, value, sub }: { icon: typeof Wallet; label: string; value: string; sub?: string }) {
  return (
    <div className="card p-4">
      <div className="flex items-center gap-2">
        <span className="grid size-8 place-items-center rounded-lg bg-brand-50 text-brand-600 dark:bg-brand-900/30"><Icon className="size-4" /></span>
        <span className="text-muted text-xs">{label}</span>
      </div>
      <div className="mt-2 truncate text-xl font-semibold">{value}</div>
      {sub && <div className="text-muted mt-0.5 text-xs">{sub}</div>}
    </div>
  )
}
