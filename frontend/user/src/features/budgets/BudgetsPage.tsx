import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ChevronLeft, ChevronRight, Pencil, PiggyBank, Plus, Trash2 } from 'lucide-react'
import { Button } from '@common/ui/Button'
import { EmptyState } from '@common/ui/EmptyState'
import { Skeleton } from '@common/ui/Skeleton'
import { Modal, ConfirmDialog } from '@common/ui/Modal'
import { Field, Input, Select } from '@common/ui/Field'
import { Alert } from '@common/ui/Alert'
import { ProgressBar } from '@common/ui/ProgressBar'
import { percentTone } from '@common/format/percent'
import { CategoryIcon } from '@common/categories/CategoryIcon'
import { formatMoney } from '@common/format/money'
import { ApiRequestError } from '@common/api/client'
import { useCategories } from '@/features/categories/useCategories'
import { analyticsKeys } from '@/features/analytics/api'
import { budgetsApi, budgetKeys, currentMonth, monthLabel, shiftMonth, type Budget } from './api'

export function BudgetsPage() {
  const qc = useQueryClient()
  const [month, setMonth] = useState(currentMonth())
  const { categories, byId } = useCategories()
  const q = useQuery({ queryKey: budgetKeys.month(month), queryFn: () => budgetsApi.list(month) })
  const [editing, setEditing] = useState<Budget | 'new' | null>(null)
  const [deleting, setDeleting] = useState<Budget | null>(null)

  const invalidate = () => {
    qc.invalidateQueries({ queryKey: budgetKeys.all })
    qc.invalidateQueries({ queryKey: analyticsKeys.summary })
  }
  const del = useMutation({
    mutationFn: (id: string) => budgetsApi.remove(id),
    onSuccess: () => { setDeleting(null); invalidate() },
  })

  const rows = q.data?.budgets ?? []
  const sum = q.data?.summary
  const currency = rows[0]?.currency ?? 'INR'
  const overall = sum && sum.totalLimit > 0 ? (sum.totalSpent * 100) / sum.totalLimit : 0
  const used = new Set(rows.map((b) => b.categoryId))

  return (
    <div className="mx-auto max-w-3xl">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Budgets</h1>
          <p className="text-muted text-sm">Monthly limits per category.</p>
        </div>
        <Button size="sm" onClick={() => setEditing('new')}><Plus className="size-4" /> Add budget</Button>
      </div>

      <div className="card mb-4 flex items-center justify-between p-2">
        <button type="button" aria-label="Previous month" onClick={() => setMonth(shiftMonth(month, -1))} className="text-muted grid size-9 place-items-center rounded-lg hover:bg-brand-50 dark:hover:bg-brand-900/30"><ChevronLeft className="size-5" /></button>
        <div className="text-sm font-semibold" data-testid="budget-month">{monthLabel(month)}</div>
        <button type="button" aria-label="Next month" onClick={() => setMonth(shiftMonth(month, 1))} className="text-muted grid size-9 place-items-center rounded-lg hover:bg-brand-50 dark:hover:bg-brand-900/30"><ChevronRight className="size-5" /></button>
      </div>

      {q.isLoading ? (
        <div className="grid gap-3"><Skeleton className="h-24" /><Skeleton className="h-20" /><Skeleton className="h-20" /></div>
      ) : rows.length === 0 ? (
        <div className="card">
          <EmptyState icon={PiggyBank} title={`No budgets for ${monthLabel(month)}`} description="Set a monthly limit for a category to track how much of it you have used." action={<Button size="sm" onClick={() => setEditing('new')}>Add budget</Button>} />
        </div>
      ) : (
        <>
          <section className="card mb-4 p-4">
            <div className="flex flex-wrap items-end justify-between gap-2">
              <div>
                <div className="text-muted text-xs">Total spent</div>
                <div className="text-2xl font-semibold">{formatMoney(sum!.totalSpent, currency)} <span className="text-muted text-sm font-normal">of {formatMoney(sum!.totalLimit, currency)}</span></div>
              </div>
              <div className={`text-sm font-semibold ${percentTone(overall)}`}>{overall.toFixed(0)}% used</div>
            </div>
            <ProgressBar percent={overall} className="mt-3" label="Overall budget" />
            <div className="text-muted mt-2 text-xs">{sum!.totalLimit - sum!.totalSpent >= 0 ? `${formatMoney(sum!.totalLimit - sum!.totalSpent, currency)} remaining this month` : `${formatMoney(sum!.totalSpent - sum!.totalLimit, currency)} over budget`}</div>
          </section>
          <ul className="grid gap-3">
            {rows.map((b) => {
              const c = byId.get(b.categoryId)
              return (
                <li key={b.id} className="card p-4">
                  <div className="flex items-center gap-3">
                    <CategoryIcon category={c} />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center justify-between gap-2">
                        <div className="truncate text-sm font-medium">{c?.name ?? 'Category'}</div>
                        <div className={`text-xs font-semibold ${percentTone(b.percent)}`}>{b.percent.toFixed(0)}%</div>
                      </div>
                      <div className="text-muted text-xs">{formatMoney(b.spent, b.currency)} of {formatMoney(b.limit, b.currency)} · {b.count} receipt{b.count === 1 ? '' : 's'}</div>
                    </div>
                    <button type="button" aria-label={`Edit ${c?.name ?? 'budget'}`} onClick={() => setEditing(b)} className="text-muted rounded-md p-1.5 hover:bg-brand-50 hover:text-brand-700 dark:hover:bg-brand-900/30"><Pencil className="size-4" /></button>
                    <button type="button" aria-label={`Delete ${c?.name ?? 'budget'}`} onClick={() => setDeleting(b)} className="text-muted rounded-md p-1.5 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950/40"><Trash2 className="size-4" /></button>
                  </div>
                  <ProgressBar percent={b.percent} className="mt-3" label={`${c?.name ?? 'Category'} budget`} />
                  <div className="text-muted mt-1.5 text-xs">{b.remaining >= 0 ? `${formatMoney(b.remaining, b.currency)} left` : `${formatMoney(-b.remaining, b.currency)} over`}</div>
                </li>
              )
            })}
          </ul>
        </>
      )}

      {editing && (
        <BudgetModal
          month={month}
          budget={editing === 'new' ? null : editing}
          categories={categories.filter((c) => editing !== 'new' ? c.id === editing.categoryId : !used.has(c.id))}
          onClose={() => setEditing(null)}
          onSaved={() => { setEditing(null); invalidate() }}
        />
      )}
      <ConfirmDialog open={!!deleting} onClose={() => setDeleting(null)} onConfirm={() => deleting && del.mutate(deleting.id)} loading={del.isPending} title="Delete budget" message={`Remove the ${byId.get(deleting?.categoryId ?? '')?.name ?? ''} budget for ${monthLabel(month)}?`} />
    </div>
  )
}

function BudgetModal({ month, budget, categories, onClose, onSaved }: {
  month: string
  budget: Budget | null
  categories: { id: string; name: string }[]
  onClose: () => void
  onSaved: () => void
}) {
  const [categoryId, setCategoryId] = useState(budget?.categoryId ?? categories[0]?.id ?? '')
  const [limit, setLimit] = useState(budget ? String(budget.limit / 100) : '')
  const [error, setError] = useState('')
  const save = useMutation({
    mutationFn: () => {
      const paise = Math.round(Number(limit) * 100)
      if (!categoryId) throw new Error('Pick a category')
      if (!Number.isFinite(paise) || paise <= 0) throw new Error('Enter a limit greater than zero')
      return budgetsApi.upsert({ categoryId, month, limit: paise })
    },
    onSuccess: onSaved,
    onError: (e) => setError(e instanceof ApiRequestError ? Object.values(e.fields ?? {})[0] ?? e.message : (e as Error).message),
  })
  return (
    <Modal open onClose={onClose} title={budget ? 'Edit budget' : 'Add budget'} footer={
      <>
        <Button variant="secondary" onClick={onClose}>Cancel</Button>
        <Button onClick={() => save.mutate()} loading={save.isPending}>{budget ? 'Save' : 'Add'}</Button>
      </>
    }>
      <form className="grid gap-3" onSubmit={(e) => { e.preventDefault(); save.mutate() }}>
        {error && <Alert>{error}</Alert>}
        <Field label="Month"><Input value={monthLabel(month)} readOnly /></Field>
        <Field label="Category">
          <Select value={categoryId} onChange={(e) => setCategoryId(e.target.value)} disabled={!!budget} name="categoryId">
            {categories.length === 0 && <option value="">All categories already have a budget</option>}
            {categories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
          </Select>
        </Field>
        <Field label="Monthly limit (₹)" hint="Whole rupees or with paise, e.g. 5000 or 4999.50">
          <Input name="limit" inputMode="decimal" placeholder="5000" value={limit} onChange={(e) => setLimit(e.target.value)} autoFocus />
        </Field>
        <button type="submit" hidden />
      </form>
    </Modal>
  )
}
