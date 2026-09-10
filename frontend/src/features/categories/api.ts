import { api } from '@/lib/api/client'
import type { Category } from './types'

export const categoriesApi = {
  list: () => api<{ categories: Category[] }>('/categories'),
  create: (body: { name: string; icon?: string; color?: string }) => api<Category>('/categories', { method: 'POST', body }),
  update: (id: string, body: { name?: string; icon?: string; color?: string }) =>
    api<Category>(`/categories/${id}`, { method: 'PATCH', body }),
  remove: (id: string) => api<void>(`/categories/${id}`, { method: 'DELETE' }),
}
