import { api } from '@common/api/client'
import type { Category } from '@common/categories/types'

export type SystemCategory = Category & { receiptCount: number; createdAt: string; updatedAt: string }
export type CategoryInput = { name?: string; slug?: string; icon?: string; color?: string }

export const categoriesApi = {
  list: () => api<{ categories: SystemCategory[] }>('/api/admin/categories'),
  create: (body: CategoryInput) => api<SystemCategory>('/api/admin/categories', { method: 'POST', body }),
  update: (id: string, body: CategoryInput) => api<SystemCategory>(`/api/admin/categories/${id}`, { method: 'PATCH', body }),
  remove: (id: string) => api<{ deleted: boolean; receiptsMoved: number }>(`/api/admin/categories/${id}`, { method: 'DELETE' }),
}

export const categoryKeys = { list: ['admin', 'categories'] as const }
