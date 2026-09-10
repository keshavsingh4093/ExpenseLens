export type KPI = { total: number; count: number; previous: number; changePct: number | null; periodLabel: string }
export type Summary = { totalExpenses: KPI; thisMonth: KPI; today: KPI; averageDaily: KPI }
export type TrendPoint = { period: string; label: string; total: number; count: number }
export type CategoryTotal = { categoryId: string | null; total: number; count: number; percent: number }
export type MerchantTotal = { merchant: string; total: number; count: number }
export type Stats = {
  total: number
  transactions: number
  avgDaily: number
  avgPerTransaction: number
  highestDay?: { date: string; total: number; count: number }
  days: number
}
export type RangeName = '7d' | '30d' | '3m' | '6m' | '1y' | 'month' | 'year' | 'all'
export type RangeParams = { range?: RangeName; from?: string; to?: string }
