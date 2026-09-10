import { api } from '@common/api/client'

export type ConfigEntry = {
  id: string
  key: string
  value: string
  description: string
  category: string
  sensitive: boolean
  createdAt: string
  updatedAt: string
}

export const configApi = {
  list: () => api<{ configs: ConfigEntry[] }>('/api/admin/system-config'),
  upsert: (key: string, body: { value: string; description?: string; category?: string; sensitive?: boolean }) =>
    api<ConfigEntry>(`/api/admin/system-config/${encodeURIComponent(key)}`, { method: 'PUT', body }),
  remove: (key: string) => api<void>(`/api/admin/system-config/${encodeURIComponent(key)}`, { method: 'DELETE' }),
  reload: () => api<{ status: string; userBackendReloaded: boolean }>('/api/admin/system-config/reload', { method: 'POST' }),
}

export const configKeys = { list: ['admin', 'system-config'] as const }
