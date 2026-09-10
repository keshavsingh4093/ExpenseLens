import { useEffect, useState } from 'react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router'
import { Search, Receipt as ReceiptIcon, ChevronLeft, ChevronRight, SlidersHorizontal, Upload } from 'lucide-react'
import { Button } from '@common/ui/Button'
import { Chip } from '@common/ui/Chip'
import { EmptyState } from '@common/ui/EmptyState'
import { Skeleton } from '@common/ui/Skeleton'
import { StatusBadge } from '@common/ui/StatusBadge'
import { Select } from '@common/ui/Field'
import { CategoryIcon, CategoryBadge } from '@common/categories/CategoryIcon'
import { useCategories } from '@/features/categories/useCategories'
import { useDebounce } from '@common/hooks/useDebounce'
import { formatMoney } from '@common/format/money'
import { formatDate, todayISO } from '@common/format/date'
import { receiptsApi, receiptKeys } from './api'
import type { ListParams } from './types'

const sorts = [['date_desc', 'Newest first'], ['date_asc', 'Oldest first'], ['amount_desc', 'Highest amount'], ['amount_asc', 'Lowest amount']] as const

export function ReceiptsPage() {
  const [sp, setSp] = useSearchParams()
  const { categories, byId } = useCategories()
  const [q, setQ] = useState(sp.get('q') ?? '')
  const dq = useDebounce(q, 300)
  const [filtersOpen, setFiltersOpen] = useState(false)

  const set = (patch: Record<string, string | undefined>) => {
    const next = new URLSearchParams(sp)
    for (const [k, v] of Object.entries(patch)) {
      if (v) next.set(k, v)
      else next.delete(k)
    }
    if (!('page' in patch)) next.delete('page')
    setSp(next, { replace: true })
  }
  useEffect(() => { if ((sp.get('q') ?? '') !== dq) set({ q: dq || undefined }) }, [dq]) // eslint-disable-line react-hooks/exhaustive-deps

  const month = sp.get('month') === '1'
  const params: ListParams = {
    q: sp.get('q') ?? undefined,
    category: sp.get('category') ?? undefined,
    status: (sp.get('status') as ListParams['status']) ?? undefined,
    sort: (sp.get('sort') as ListParams['sort']) ?? 'date_desc',
    page: Number(sp.get('page') ?? 1),
    pageSize: 20,
    ...(month ? { from: todayISO().slice(0, 7) + '-01', to: todayISO() } : {}),
  }
  const list = useQuery({ queryKey: receiptKeys.list(params), queryFn: () => receiptsApi.list(params), placeholderData: keepPreviousData })
  const pages = Math.max(1, Math.ceil((list.data?.total ?? 0) / 20))
  const cat = sp.get('category')
  const anyFilter = !!(cat || sp.get('status') || month)

  return (
    <div className="mx-auto max-w-3xl">
      <div className="mb-4 flex items-center justify-between gap-3">
        <h1 className="text-2xl font-semibold">Receipts</h1>
        <Link to="/upload"><Button size="sm"><Upload className="size-4" /> Upload</Button></Link>
      </div>

      <div className="relative mb-3">
        <Search className="text-muted absolute top-1/2 left-3 size-4 -translate-y-1/2" />
        <input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Search receipts, merchants, items…"
          className="h-10 w-full rounded-lg border border-[var(--border)] bg-[var(--surface)] pr-3 pl-9 text-sm outline-none focus:border-brand-500 focus:ring-2 focus:ring-brand-500/30"
        />
      </div>

      <div className="mb-4 flex gap-2 overflow-x-auto pb-1">
        <Chip active={!anyFilter} onClick={() => set({ category: undefined, status: undefined, month: undefined })}>All</Chip>
        <Chip active={month} onClick={() => set({ month: month ? undefined : '1' })}>This Month</Chip>
        <Chip active={sp.get('status') === 'review'} onClick={() => set({ status: sp.get('status') === 'review' ? undefined : 'review' })}>Needs review</Chip>
        <Chip active={!!cat || filtersOpen} onClick={() => setFiltersOpen((o) => !o)}><SlidersHorizontal className="mr-1 inline size-3" />{cat ? byId.get(cat)?.name ?? 'Category' : 'Category'}</Chip>
        <div className="ml-auto shrink-0">
          <Select value={params.sort} onChange={(e) => set({ sort: e.target.value })} className="h-8 w-auto! text-xs" aria-label="Sort">
            {sorts.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
          </Select>
        </div>
      </div>
      {filtersOpen && (
        <div className="mb-4 flex flex-wrap gap-2">
          {categories.map((c) => (
            <Chip key={c.id} active={cat === c.id} onClick={() => set({ category: cat === c.id ? undefined : c.id })}>{c.name}</Chip>
          ))}
        </div>
      )}

      {list.isLoading ? (
        <div className="grid gap-2">{Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} className="h-16" />)}</div>
      ) : list.data && list.data.receipts.length > 0 ? (
        <>
          <ul className="card divide-y divide-[var(--border)]">
            {list.data.receipts.map((r) => {
              const c = r.categoryId ? byId.get(r.categoryId) : undefined
              const to = r.status === 'confirmed' || r.status === 'failed' ? `/receipts/${r.id}` : `/receipts/${r.id}/review`
              return (
                <li key={r.id}>
                  <Link to={to} className="flex items-center gap-3 px-3 py-3 hover:bg-brand-50/50 dark:hover:bg-brand-900/10">
                    {r.thumbnail && !r.thumbnail.endsWith('.pdf') ? (
                      <img src={r.thumbnail} alt="" className="size-10 shrink-0 rounded-lg object-cover" loading="lazy" />
                    ) : (
                      <CategoryIcon category={c} />
                    )}
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="truncate font-medium">{r.merchant || 'Unknown merchant'}</span>
                        {r.status !== 'confirmed' && <StatusBadge status={r.status} />}
                      </div>
                      <div className="text-muted mt-0.5 flex items-center gap-2 text-xs">
                        <span>{r.date ? formatDate(r.date) : formatDate(r.createdAt)}</span>
                        <CategoryBadge category={c} />
                      </div>
                    </div>
                    <div className="text-right text-sm font-semibold tabular-nums">{formatMoney(r.total, r.currency)}</div>
                  </Link>
                </li>
              )
            })}
          </ul>
          {pages > 1 && (
            <div className="mt-4 flex items-center justify-between text-sm">
              <Button variant="secondary" size="sm" disabled={params.page! <= 1} onClick={() => set({ page: String(params.page! - 1) })}><ChevronLeft className="size-4" /> Prev</Button>
              <span className="text-muted">Page {params.page} of {pages} · {list.data.total} receipts</span>
              <Button variant="secondary" size="sm" disabled={params.page! >= pages} onClick={() => set({ page: String(params.page! + 1) })}>Next <ChevronRight className="size-4" /></Button>
            </div>
          )}
        </>
      ) : (
        <div className="card">
          <EmptyState
            icon={ReceiptIcon}
            title={params.q || anyFilter ? 'No receipts match' : 'No receipts yet'}
            description={params.q || anyFilter ? 'Try a different search or clear the filters.' : 'Upload your first receipt and we will extract the details for you.'}
            action={!params.q && !anyFilter ? <Link to="/upload"><Button size="sm">Upload receipt</Button></Link> : undefined}
          />
        </div>
      )}
    </div>
  )
}
