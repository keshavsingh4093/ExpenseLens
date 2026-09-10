import type { ReceiptStatus } from '@common/types'
export type { ReceiptStatus }

export type ReceiptItem = { id?: string; name: string; quantity?: number; unitPrice?: number; amount: number }
export type ReceiptImage = { id: string; url: string; contentType: string; size: number; order: number; fileName?: string }
export type TaxLine = { label: string; amount: number }

export type Receipt = {
  id: string
  status: ReceiptStatus
  merchant: string
  date: string
  time?: string
  categoryId?: string
  currency: string
  subtotal?: number
  tax?: number
  total: number
  paymentMethod?: string
  gstin?: string
  taxBreakdown?: TaxLine[]
  items: ReceiptItem[]
  images: ReceiptImage[]
  notes: string
  tags: string[]
  extracted?: {
    model: string
    confidence: number
    suggestedCategory?: string
    warrantyCandidates?: { itemName: string; reason: string }[]
    extractedAt: string
    error?: string
    attempts: number
  }
  createdAt: string
  updatedAt: string
  confirmedAt?: string
}

export type ReceiptListItem = {
  id: string
  status: ReceiptStatus
  merchant: string
  date: string
  categoryId?: string
  currency: string
  total: number
  itemCount: number
  thumbnail?: string
  createdAt: string
}

export type ReceiptPatch = Partial<{
  merchant: string
  date: string
  time: string
  categoryId: string
  currency: string
  subtotal: number
  tax: number
  total: number
  paymentMethod: string
  gstin: string
  items: ReceiptItem[]
  notes: string
  tags: string[]
}>

export type UploadResult = { id: string; status: ReceiptStatus; duplicate: boolean; fileName: string }

export type ListParams = Partial<{
  q: string
  category: string
  status: ReceiptStatus
  from: string
  to: string
  sort: 'date_desc' | 'date_asc' | 'amount_desc' | 'amount_asc' | 'created_desc'
  page: number
  pageSize: number
}>

export const paymentMethods = [
  ['upi', 'UPI'], ['card', 'Card'], ['cash', 'Cash'], ['netbanking', 'Net banking'], ['wallet', 'Wallet'], ['other', 'Other'],
] as const
