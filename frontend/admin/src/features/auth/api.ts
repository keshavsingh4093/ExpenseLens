import { api } from '@common/api/client'
import type { Admin, AdminSession } from './types'

type Wire = { accessToken: string; admin: Admin }
const toSession = (w: Wire): AdminSession => ({ accessToken: w.accessToken, user: w.admin })

export const adminAuthApi = {
  login: (body: { email: string; password: string }) => api<Wire>('/openapi/admin/login', { method: 'POST', body }).then(toSession),
  refresh: () => api<Wire>('/openapi/admin/auth/refresh', { method: 'POST', retry: false }).then(toSession),
  logout: () => api<void>('/openapi/admin/auth/logout', { method: 'POST' }),
  me: () => api<Admin>('/api/admin/me'),
}
