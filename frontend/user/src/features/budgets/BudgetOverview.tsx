import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router'
import { PiggyBank } from 'lucide-react'
import { Button } from '@common/ui/Button'
import { EmptyState } from '@common/ui/EmptyState'
import { Skeleton } from '@common/ui/Skeleton'
import { ProgressBar } from '@common/ui/ProgressBar'
import { percentTone } from '@common/format/percent'
import { CategoryIcon } from '@common/categories/CategoryIcon'
import { formatMoney } from '@common/format/money'
import { useCategories } from '@/features/categories/useCategories'
import { budgetsApi, budgetKeys, currentMonth } from './api'

/** Dashboard card: the three most-used budgets of the current month. */
export function BudgetOverview() {
  const month = currentMonth()
  const { byId } = useCategories()
  const q = useQuery({ queryKey: budgetKeys.month(month), queryFn: () => budgetsApi.list(month) })
  if (q.isLoading) return <div className="grid gap-3"><Skeleton className="h-10" /><Skeleton className="h-10" /><Skeleton className="h-10" /></div>
  const rows = [...(q.data?.budgets ?? [])].sort((a, b) => b.percent - a.percent).slice(0, 3)
  if (rows.length === 0) {
    return <EmptyState icon={PiggyBank} title="No budgets yet" description="Set monthly limits per category to track progress here." action={<Link to="/budgets"><Button size="sm" variant="secondary">Set up budgets</Button></Link>} />
  }
  return (
    <ul className="grid gap-3">
      {rows.map((b) => {
        const c = byId.get(b.categoryId)
        return (
          <li key={b.id}>
            <div className="flex items-center gap-2">
              <CategoryIcon category={c} size="sm" />
              <span className="min-w-0 flex-1 truncate text-sm font-medium">{c?.name ?? 'Category'}</span>
              <span className="text-muted text-xs tabular-nums">{formatMoney(b.spent, b.currency)} / {formatMoney(b.limit, b.currency)}</span>
              <span className={`w-10 text-right text-xs font-semibold ${percentTone(b.percent)}`}>{b.percent.toFixed(0)}%</span>
            </div>
            <ProgressBar percent={b.percent} className="mt-1.5" label={`${c?.name ?? 'Category'} budget`} />
          </li>
        )
      })}
    </ul>
  )
}
