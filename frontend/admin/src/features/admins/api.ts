import { api } from '@common/api/client'
import type { Admin } from '@/features/auth/types'
import { qs } from '@/features/users/api'

export type AdminRole = Admin['role']
export type AdminsParams = { q?: string; page?: number; pageSize?: number }
export type AdminsPage = { admins: Admin[]; total: number; page: number; pageSize: number }
export type CreateAdminInput = { email: string; name: string; password: string; role: AdminRole }

export const adminsApi = {
  list: (params: AdminsParams) => api<AdminsPage>(`/api/admin/admins${qs(params)}`),
  create: (body: CreateAdminInput) => api<{ admin: Admin }>('/api/admin/admins', { method: 'POST', body }),
  update: (id: string, body: { name?: string; role?: AdminRole }) => api<{ admin: Admin }>(`/api/admin/admins/${id}`, { method: 'PATCH', body }),
  disable: (id: string) => api<{ admin: Admin }>(`/api/admin/admins/${id}/disable`, { method: 'POST' }),
  enable: (id: string) => api<{ admin: Admin }>(`/api/admin/admins/${id}/enable`, { method: 'POST' }),
  resetPassword: (id: string, password: string) =>
    api<{ admin: Admin }>(`/api/admin/admins/${id}/reset-password`, { method: 'POST', body: { password } }),
}

export const adminKeys = {
  all: ['admin', 'admins'] as const,
  list: (params: AdminsParams) => ['admin', 'admins', params] as const,
}

/** 16 characters from an unambiguous alphabet, via the Web Crypto API. */
export function generatePassword(): string {
  const alphabet = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789'
  const bytes = new Uint8Array(16)
  crypto.getRandomValues(bytes)
  return Array.from(bytes, (b) => alphabet[b % alphabet.length]).join('')
}
