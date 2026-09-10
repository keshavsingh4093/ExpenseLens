import { api } from '@common/api/client'

export type Budget = {
  id: string
  categoryId: string
  month: string
  limit: number
  currency: string
  spent: number
  remaining: number
  percent: number
  count: number
}
export type BudgetsResponse = { month: string; budgets: Budget[]; summary: { totalLimit: number; totalSpent: number } }

const tz = () => Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Kolkata'

export const budgetsApi = {
  list: (month?: string) => api<BudgetsResponse>(`/api/budgets?tz=${encodeURIComponent(tz())}${month ? `&month=${month}` : ''}`),
  upsert: (body: { categoryId: string; month: string; limit: number }) => api<Budget>('/api/budgets', { method: 'PUT', body }),
  remove: (id: string) => api<void>(`/api/budgets/${id}`, { method: 'DELETE' }),
}

export const budgetKeys = {
  all: ['budgets'] as const,
  month: (m: string) => ['budgets', m] as const,
}

/** YYYY-MM for today in local time. */
export function currentMonth(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
}

export function shiftMonth(month: string, delta: number): string {
  const [y, m] = month.split('-').map(Number)
  const d = new Date(y, m - 1 + delta, 1)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
}

export function monthLabel(month: string, locale = 'en-IN'): string {
  const [y, m] = month.split('-').map(Number)
  return new Intl.DateTimeFormat(locale, { month: 'long', year: 'numeric' }).format(new Date(y, m - 1, 1))
}
