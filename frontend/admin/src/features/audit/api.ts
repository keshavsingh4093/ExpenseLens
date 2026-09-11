import { api } from '@common/api/client'
import { qs } from '@/features/users/api'

export type AuditEntry = {
  id: string
  adminId: string
  adminEmail: string
  action: string
  targetType: string
  targetId: string
  details?: Record<string, unknown>
  ip?: string
  at: string
}

export type AuditParams = { action?: string; adminId?: string; targetId?: string; page?: number; pageSize?: number }

export const auditApi = {
  list: (params: AuditParams) => api<{ entries: AuditEntry[]; total: number; page: number; pageSize: number }>(`/api/admin/audit${qs(params)}`),
  actions: () => api<{ actions: string[] }>('/api/admin/audit/actions'),
}

export const auditKeys = {
  list: (params: AuditParams) => ['admin', 'audit', params] as const,
  actions: ['admin', 'audit', 'actions'] as const,
}
