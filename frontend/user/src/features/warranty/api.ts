import { api } from '@common/api/client'

export type WarrantyStatus = 'active' | 'expiring' | 'expired'
export type Warranty = {
  id: string
  productName: string
  purchaseDate: string
  expiryDate: string
  durationMonths?: number
  receiptId?: string
  itemId?: string
  imageUrl?: string
  notes: string
  status: WarrantyStatus
  daysLeft: number
  createdAt: string
}
export type WarrantyInput = Partial<{
  productName: string
  purchaseDate: string
  expiryDate: string
  durationMonths: number
  receiptId: string
  itemId: string
  notes: string
}>

export const warrantyApi = {
  list: (status?: WarrantyStatus) =>
    api<{ warranties: Warranty[]; counts: Record<WarrantyStatus, number> }>(`/api/warranties${status ? `?status=${status}` : ''}`),
  create: (body: WarrantyInput) => api<Warranty>('/api/warranties', { method: 'POST', body }),
  update: (id: string, body: WarrantyInput) => api<Warranty>(`/api/warranties/${id}`, { method: 'PATCH', body }),
  remove: (id: string) => api<void>(`/api/warranties/${id}`, { method: 'DELETE' }),
}

export const warrantyKeys = { all: ['warranties'] as const }
