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

export type UserStatusFilter = '' | 'active' | 'disabled'
export type UsersParams = { q?: string; status?: UserStatusFilter; page?: number; pageSize?: number }
export type UsersPage = { users: PortalUser[]; total: number; page: number; pageSize: number }

export function qs(params: Record<string, unknown>) {
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) if (v !== undefined && v !== '' && v !== null) p.set(k, String(v))
  const s = p.toString()
  return s ? `?${s}` : ''
}

export const usersApi = {
  list: (params: UsersParams) => api<UsersPage>(`/api/admin/users${qs(params)}`),
  get: (id: string) => api<{ user: PortalUser; receiptCount: number }>(`/api/admin/users/${id}`),
  disable: (id: string) => api<{ user: PortalUser }>(`/api/admin/users/${id}/disable`, { method: 'POST' }),
  enable: (id: string) => api<{ user: PortalUser }>(`/api/admin/users/${id}/enable`, { method: 'POST' }),
  resetLink: (id: string) => api<{ link: string; expiresAt: string }>(`/api/admin/users/${id}/reset-password-link`, { method: 'POST' }),
  remove: (id: string) => api<void>(`/api/admin/users/${id}`, { method: 'DELETE' }),
}

export const userKeys = {
  all: ['admin', 'users'] as const,
  list: (params: UsersParams) => ['admin', 'users', params] as const,
  detail: (id: string) => ['admin', 'users', 'detail', id] as const,
}
