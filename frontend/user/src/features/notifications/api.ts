import { api } from '@common/api/client'

export type NotificationType = 'budget_alert' | 'warranty_expiring' | 'extraction_done' | 'extraction_failed' | 'weekly_summary'
export type Notification = {
  id: string
  type: NotificationType
  title: string
  body: string
  data?: Record<string, unknown>
  readAt?: string
  createdAt: string
}
export type NotificationsPage = { notifications: Notification[]; total: number; unread: number; page: number; pageSize: number }

export const notificationsApi = {
  list: (page = 1, pageSize = 20) => api<NotificationsPage>(`/api/notifications?page=${page}&pageSize=${pageSize}`),
  unreadCount: () => api<{ unread: number }>('/api/notifications/unread-count'),
  read: (id: string) => api<Notification>(`/api/notifications/${id}/read`, { method: 'POST' }),
  readAll: () => api<{ updated: number }>('/api/notifications/read-all', { method: 'POST' }),
}

export const notificationKeys = {
  all: ['notifications'] as const,
  list: (page: number) => ['notifications', 'list', page] as const,
  unread: ['notifications', 'unread'] as const,
}

/** Where a notification takes you when clicked. */
export function notificationLink(n: Notification): string | null {
  const d = n.data ?? {}
  if (typeof d.receiptId === 'string') return n.type === 'extraction_done' && d.status === 'review' ? `/receipts/${d.receiptId}/review` : `/receipts/${d.receiptId}`
  if (n.type === 'budget_alert') return '/budgets'
  if (n.type === 'warranty_expiring') return '/warranty'
  if (n.type === 'weekly_summary') return '/analytics'
  return null
}
