import { api } from '@common/api/client'

export type PortalUser = {
  id: string
  email: string
  emailVerified: boolean
  name: string
  avatarUrl?: string
  hasPassword: boolean
  providers: string[]
  preferences: { currency: string; language: string; theme: string; timezone: string }
  disabled?: boolean
  createdAt: string
}

export type UsersPage = { users: PortalUser[]; total: number; page: number; pageSize: number }

function qs(params: Record<string, unknown>) {
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) if (v !== undefined && v !== '' && v !== null) p.set(k, String(v))
  const s = p.toString()
  return s ? `?${s}` : ''
}

export const usersApi = {
  list: (params: { q?: string; page?: number; pageSize?: number }) => api<UsersPage>(`/api/admin/users${qs(params)}`),
  get: (id: string) => api<{ user: PortalUser; receiptCount: number }>(`/api/admin/users/${id}`),
}

export const userKeys = {
  list: (params: { q?: string; page?: number; pageSize?: number }) => ['admin', 'users', params] as const,
  detail: (id: string) => ['admin', 'users', 'detail', id] as const,
}
