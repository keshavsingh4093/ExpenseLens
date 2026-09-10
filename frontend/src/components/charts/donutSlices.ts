import type { CategoryTotal } from '@/features/analytics/types'
import type { Category } from '@/features/categories/types'
import { categoryColor } from '@/features/categories/colors'

export type DonutSlice = { id: string; name: string; total: number; percent: number; color: string }

/** Folds beyond 6 named slices into "Other" so the ring never needs a 9th hue. */
export function toSlices(rows: CategoryTotal[], byId: Map<string, Category>, max = 6): DonutSlice[] {
  const named = rows.map((r) => {
    const c = r.categoryId ? byId.get(r.categoryId) : undefined
    return { id: r.categoryId ?? 'none', name: c?.name ?? 'Uncategorized', total: r.total, percent: r.percent, color: categoryColor(c) }
  })
  if (named.length <= max) return named
  const head = named.slice(0, max - 1)
  const rest = named.slice(max - 1)
  head.push({ id: 'other', name: 'Other', total: rest.reduce((s, r) => s + r.total, 0), percent: rest.reduce((s, r) => s + r.percent, 0), color: '#8a8f99' })
  return head
}

