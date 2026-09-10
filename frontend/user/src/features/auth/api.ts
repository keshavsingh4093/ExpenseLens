import { api } from '@common/api/client'
import type { Preferences, Session, User } from './types'

/** Unauthenticated session routes live under /openapi/auth; profile routes under /api/me. */
export const authApi = {
  register: (body: { email: string; name: string; password: string }) => api<Session>('/openapi/auth/register', { method: 'POST', body }),
  login: (body: { email: string; password: string }) => api<Session>('/openapi/auth/login', { method: 'POST', body }),
  logout: () => api<void>('/openapi/auth/logout', { method: 'POST' }),
  refresh: () => api<Session>('/openapi/auth/refresh', { method: 'POST', retry: false }),
  me: () => api<User>('/api/me'),
  forgotPassword: (email: string) => api<{ message: string }>('/openapi/auth/forgot-password', { method: 'POST', body: { email } }),
  resetPassword: (token: string, password: string) => api<void>('/openapi/auth/reset-password', { method: 'POST', body: { token, password } }),
  changePassword: (body: { currentPassword: string; newPassword: string }) => api<void>('/api/me/password', { method: 'POST', body }),
  providers: () => api<{ providers: string[] }>('/openapi/auth/providers'),
  oauthStartUrl: (provider: string) => `/openapi/auth/${provider}/start`,
}

export const meApi = {
  update: (body: { name?: string; avatarUrl?: string }) => api<User>('/api/me', { method: 'PATCH', body }),
  updatePreferences: (body: Partial<Omit<Preferences, 'notifications'>> & { notifications?: Partial<Preferences['notifications']> }) =>
    api<User>('/api/me/preferences', { method: 'PATCH', body }),
  deleteAccount: () => api<void>('/api/me', { method: 'DELETE' }),
}
