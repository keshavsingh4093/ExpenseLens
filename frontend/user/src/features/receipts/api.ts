import { api } from '@common/api/client'
import type { ListParams, Receipt, ReceiptListItem, ReceiptPatch, UploadResult } from './types'

function qs(params: Record<string, unknown>) {
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) if (v !== undefined && v !== '' && v !== null) p.set(k, String(v))
  const s = p.toString()
  return s ? `?${s}` : ''
}

export const receiptsApi = {
  upload: (files: File[], group = false) => {
    const fd = new FormData()
    for (const f of files) fd.append('files', f, f.name)
    if (group) fd.append('group', 'true')
    return api<{ receipts: UploadResult[] }>('/api/receipts/upload', { method: 'POST', body: fd })
  },
  list: (params: ListParams) =>
    api<{ receipts: ReceiptListItem[]; total: number; page: number; pageSize: number }>(`/api/receipts${qs(params)}`),
  recent: (limit = 5) => api<{ receipts: Receipt[] }>(`/api/receipts/recent-uploads?limit=${limit}`),
  get: (id: string) => api<Receipt>(`/api/receipts/${id}`),
  update: (id: string, patch: ReceiptPatch) => api<Receipt>(`/api/receipts/${id}`, { method: 'PATCH', body: patch }),
  confirm: (id: string, patch: ReceiptPatch) => api<Receipt>(`/api/receipts/${id}/confirm`, { method: 'POST', body: patch }),
  reextract: (id: string) => api<Receipt>(`/api/receipts/${id}/reextract`, { method: 'POST' }),
  remove: (id: string) => api<void>(`/api/receipts/${id}`, { method: 'DELETE' }),
  addImages: (id: string, files: File[]) => {
    const fd = new FormData()
    for (const f of files) fd.append('files', f, f.name)
    return api<Receipt>(`/api/receipts/${id}/images`, { method: 'POST', body: fd })
  },
  removeImage: (id: string, imageId: string) => api<Receipt>(`/api/receipts/${id}/images/${imageId}`, { method: 'DELETE' }),
}

export const receiptKeys = {
  all: ['receipts'] as const,
  list: (params: ListParams) => ['receipts', 'list', params] as const,
  recent: ['receipts', 'recent'] as const,
  detail: (id: string) => ['receipts', 'detail', id] as const,
}
