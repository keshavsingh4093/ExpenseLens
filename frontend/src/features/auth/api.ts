import { api } from '@/lib/api/client'
import type { Preferences, Session, User } from './types'

export const authApi = {
  register: (body: { email: string; name: string; password: string }) => api<Session>('/auth/register', { method: 'POST', body }),
  login: (body: { email: string; password: string }) => api<Session>('/auth/login', { method: 'POST', body }),
  logout: () => api<void>('/auth/logout', { method: 'POST' }),
  refresh: () => api<Session>('/auth/refresh', { method: 'POST', retry: false }),
  me: () => api<User>('/auth/me'),
  forgotPassword: (email: string) => api<{ message: string }>('/auth/forgot-password', { method: 'POST', body: { email } }),
  resetPassword: (token: string, password: string) => api<void>('/auth/reset-password', { method: 'POST', body: { token, password } }),
  changePassword: (body: { currentPassword: string; newPassword: string }) => api<void>('/auth/password', { method: 'POST', body }),
  providers: () => api<{ providers: string[] }>('/auth/providers'),
  oauthStartUrl: (provider: string) => `/api/v1/auth/${provider}/start`,
}

export const meApi = {
  update: (body: { name?: string; avatarUrl?: string }) => api<User>('/me', { method: 'PATCH', body }),
  updatePreferences: (body: Partial<Omit<Preferences, 'notifications'>> & { notifications?: Partial<Preferences['notifications']> }) =>
    api<User>('/me/preferences', { method: 'PATCH', body }),
  deleteAccount: () => api<void>('/me', { method: 'DELETE' }),
}
