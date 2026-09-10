import { api } from '@common/api/client'

export type Overview = {
  users: { total: number; last7Days: number; last30Days: number; withReceipts: number }
  receipts: { total: number; last7Days: number; last30Days: number; byStatus: Record<string, number>; failed: number; failurePct: number }
  generatedAt: string
}

export const statsApi = { overview: () => api<Overview>('/api/admin/stats') }
export const statsKeys = { overview: ['admin', 'stats'] as const }
