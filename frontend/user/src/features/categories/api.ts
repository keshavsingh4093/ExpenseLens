import { api } from '@common/api/client'
import type { Category } from '@common/categories/types'

export const categoriesApi = {
  list: () => api<{ categories: Category[] }>('/api/categories'),
  create: (body: { name: string; icon?: string; color?: string }) => api<Category>('/api/categories', { method: 'POST', body }),
  update: (id: string, body: { name?: string; icon?: string; color?: string }) =>
    api<Category>(`/api/categories/${id}`, { method: 'PATCH', body }),
  remove: (id: string) => api<void>(`/api/categories/${id}`, { method: 'DELETE' }),
}
