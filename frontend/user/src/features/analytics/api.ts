import { api } from '@common/api/client'
import type { CategoryTotal, MerchantTotal, RangeParams, Stats, Summary, TrendPoint } from '@common/analytics/types'

const tz = () => Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Kolkata'

function qs(params: Record<string, unknown>) {
  const p = new URLSearchParams({ tz: tz() })
  for (const [k, v] of Object.entries(params)) if (v !== undefined && v !== '') p.set(k, String(v))
  return `?${p.toString()}`
}

export const analyticsApi = {
  summary: () => api<Summary>(`/api/analytics/summary${qs({})}`),
  trend: (r: RangeParams & { granularity?: 'day' | 'week' | 'month' }) =>
    api<{ granularity: string; points: TrendPoint[] }>(`/api/analytics/trend${qs(r)}`),
  categories: (r: RangeParams) => api<{ categories: CategoryTotal[] }>(`/api/analytics/categories${qs(r)}`),
  merchants: (r: RangeParams & { limit?: number }) => api<{ merchants: MerchantTotal[] }>(`/api/analytics/merchants${qs(r)}`),
  stats: (r: RangeParams) => api<{ stats: Stats }>(`/api/analytics/stats${qs(r)}`),
}

export const analyticsKeys = {
  summary: ['analytics', 'summary'] as const,
  trend: (r: RangeParams) => ['analytics', 'trend', r] as const,
  categories: (r: RangeParams) => ['analytics', 'categories', r] as const,
  merchants: (r: RangeParams & { limit?: number }) => ['analytics', 'merchants', r] as const,
  stats: (r: RangeParams) => ['analytics', 'stats', r] as const,
}
