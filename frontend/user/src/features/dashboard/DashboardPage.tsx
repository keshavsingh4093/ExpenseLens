import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router'
import { Wallet, CalendarDays, Sun, Gauge, Upload, Receipt as ReceiptIcon, PiggyBank } from 'lucide-react'
import { Button } from '@common/ui/Button'
import { Skeleton } from '@common/ui/Skeleton'
import { EmptyState } from '@common/ui/EmptyState'
import { Select } from '@common/ui/Field'
import { StatTile } from '@common/charts/StatTile'
import { TrendChart } from '@common/charts/TrendChart'
import { CategoryDonut } from '@common/charts/CategoryDonut'
import { toSlices } from '@common/charts/donutSlices'
import { analyticsApi, analyticsKeys } from '@/features/analytics/api'
import type { RangeName } from '@common/analytics/types'
import { useCategories } from '@/features/categories/useCategories'
import { CategoryIcon, CategoryBadge } from '@common/categories/CategoryIcon'
import { receiptsApi, receiptKeys } from '@/features/receipts/api'
import { useAuth } from '@/features/auth/useAuth'
import { formatMoney } from '@common/format/money'
import { formatDate } from '@common/format/date'

const trendRanges: [RangeName, string][] = [['30d', 'Last 30 days'], ['3m', 'Last 3 months'], ['6m', 'Last 6 months'], ['year', 'This Year']]

export function DashboardPage() {
  const { user } = useAuth()
  const currency = user?.preferences.currency ?? 'INR'
  const { byId } = useCategories()
  const [range, setRange] = useState<RangeName>('30d')

  const summary = useQuery({ queryKey: analyticsKeys.summary, queryFn: analyticsApi.summary })
  const trend = useQuery({ queryKey: analyticsKeys.trend({ range }), queryFn: () => analyticsApi.trend({ range }) })
  const cats = useQuery({ queryKey: analyticsKeys.categories({ range: 'month' }), queryFn: () => analyticsApi.categories({ range: 'month' }) })
  const recent = useQuery({ queryKey: receiptKeys.list({ page: 1, pageSize: 4 }), queryFn: () => receiptsApi.list({ page: 1, pageSize: 4, sort: 'created_desc' }) })

  const firstName = user?.name.split(' ')[0] ?? ''
  const s = summary.data
  const slices = cats.data ? toSlices(cats.data.categories, byId) : []
  const catTotal = cats.data?.categories.reduce((a, c) => a + c.total, 0) ?? 0
  const hasTrend = trend.data?.points.some((p) => p.total > 0)

  return (
    <div className="mx-auto max-w-6xl">
      <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Hello, {firstName} 👋</h1>
          <p className="text-muted text-sm">Track your expenses and save more!</p>
        </div>
        <Link to="/upload"><Button><Upload className="size-4" /> Upload Receipt</Button></Link>
      </div>

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        {summary.isLoading || !s ? (
          Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-28" />)
        ) : (
          <>
            <StatTile label="Total Expenses" value={formatMoney(s.totalExpenses.total, currency)} delta={s.thisMonth.changePct} periodLabel={s.thisMonth.periodLabel} icon={Wallet} />
            <StatTile label="This Month" value={formatMoney(s.thisMonth.total, currency)} delta={s.thisMonth.changePct} periodLabel={s.thisMonth.periodLabel} icon={CalendarDays} />
            <StatTile label="Today" value={formatMoney(s.today.total, currency)} delta={s.today.changePct} periodLabel={s.today.periodLabel} icon={Sun} />
            <StatTile label="Average Daily" value={formatMoney(s.averageDaily.total, currency)} delta={s.averageDaily.changePct} periodLabel={s.averageDaily.periodLabel} icon={Gauge} />
          </>
        )}
      </div>

      <div className="mt-4 grid gap-4 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
        <section className="card p-4">
          <div className="mb-2 flex items-center justify-between">
            <h2 className="text-sm font-semibold">Expense Trend</h2>
            <Select value={range} onChange={(e) => setRange(e.target.value as RangeName)} className="h-8 w-auto! text-xs" aria-label="Trend range">
              {trendRanges.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
            </Select>
          </div>
          {trend.isLoading ? <Skeleton className="h-[220px]" /> : hasTrend ? (
            <TrendChart points={trend.data!.points} currency={currency} />
          ) : (
            <EmptyState icon={ReceiptIcon} title="No spending in this period" description="Upload receipts to see your trend." />
          )}
        </section>

        <section className="card p-4">
          <h2 className="mb-3 text-sm font-semibold">Category Breakdown <span className="text-muted font-normal">· this month</span></h2>
          {cats.isLoading ? <Skeleton className="h-[180px]" /> : slices.length ? (
            <CategoryDonut slices={slices} total={catTotal} currency={currency} />
          ) : (
            <EmptyState icon={ReceiptIcon} title="Nothing categorized yet" description="Confirmed receipts show up here by category." />
          )}
        </section>

        <section className="card p-4">
          <div className="mb-2 flex items-center justify-between">
            <h2 className="text-sm font-semibold">Recent Receipts</h2>
            <Link to="/receipts" className="text-xs font-medium text-brand-700 hover:underline">View All</Link>
          </div>
          {recent.isLoading ? (
            <div className="grid gap-2"><Skeleton className="h-12" /><Skeleton className="h-12" /><Skeleton className="h-12" /></div>
          ) : recent.data?.receipts.length ? (
            <ul className="divide-y divide-[var(--border)]">
              {recent.data.receipts.map((r) => {
                const c = r.categoryId ? byId.get(r.categoryId) : undefined
                return (
                  <li key={r.id}>
                    <Link to={r.status === 'confirmed' ? `/receipts/${r.id}` : `/receipts/${r.id}/review`} className="flex items-center gap-3 py-2.5 hover:bg-brand-50/50 dark:hover:bg-brand-900/10">
                      <CategoryIcon category={c} size="sm" />
                      <div className="min-w-0 flex-1">
                        <div className="truncate text-sm font-medium">{r.merchant || 'Unknown merchant'}</div>
                        <div className="text-muted text-xs">{formatDate(r.date || r.createdAt)}</div>
                      </div>
                      <CategoryBadge category={c} />
                      <div className="w-20 text-right text-sm font-semibold tabular-nums">{formatMoney(r.total, r.currency)}</div>
                    </Link>
                  </li>
                )
              })}
            </ul>
          ) : (
            <EmptyState icon={ReceiptIcon} title="No receipts yet" action={<Link to="/upload"><Button size="sm">Upload receipt</Button></Link>} />
          )}
        </section>

        <section className="card p-4">
          <div className="mb-2 flex items-center justify-between">
            <h2 className="text-sm font-semibold">Budget Overview</h2>
            <Link to="/budgets" className="text-xs font-medium text-brand-700 hover:underline">View All</Link>
          </div>
          <EmptyState icon={PiggyBank} title="No budgets yet" description="Set monthly limits per category to track progress here." action={<Link to="/budgets"><Button size="sm" variant="secondary">Set up budgets</Button></Link>} />
        </section>
      </div>
    </div>
  )
}
