import { api } from '@common/api/client'
import { qs } from '@/features/users/api'

export type ExtractionRange = '7d' | '30d' | '90d'
export type ExtractionDay = { date: string; total: number; failed: number; processing: number }
export type ExtractionModel = { model: string; total: number; failed: number; avgConfidence: number }
export type ExtractionStats = {
  range: ExtractionRange
  from: string
  totals: { total: number; failed: number; processing: number; succeeded: number; avgConfidence: number; avgAttempts: number }
  failurePct: number
  byDay: ExtractionDay[]
  byModel: ExtractionModel[]
  extractor: string
  model: string
  generatedAt: string
}
export type FailedReceipt = {
  id: string
  userId: string
  userEmail: string
  merchant: string
  imageCount: number
  error: string
  model: string
  attempts: number
  failedAt: string
  createdAt: string
}

export const extractionApi = {
  stats: (range: ExtractionRange) => api<ExtractionStats>(`/api/admin/extraction/stats${qs({ range })}`),
  failed: (page: number, pageSize: number) =>
    api<{ receipts: FailedReceipt[]; total: number; page: number; pageSize: number }>(`/api/admin/extraction/failed${qs({ page, pageSize })}`),
  retry: (id: string) => api<{ id: string; status: string }>(`/api/admin/extraction/${id}/retry`, { method: 'POST' }),
}

export const extractionKeys = {
  all: ['admin', 'extraction'] as const,
  stats: (range: ExtractionRange) => ['admin', 'extraction', 'stats', range] as const,
  failed: (page: number) => ['admin', 'extraction', 'failed', page] as const,
}
