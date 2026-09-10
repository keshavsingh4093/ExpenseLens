import { z } from 'zod'
import type { Receipt, ReceiptPatch } from './types'

const money = z.string().refine((v) => v === '' || /^\d+(\.\d{1,2})?$/.test(v), 'Enter an amount like 1230.50')
export const receiptSchema = z.object({
  merchant: z.string().trim().min(1, 'Merchant is required').max(120),
  date: z.string().regex(/^\d{4}-\d{2}-\d{2}$/, 'Pick a date'),
  time: z.string().max(8),
  categoryId: z.string(),
  currency: z.string().length(3),
  total: money.refine((v) => v !== '', 'Total is required'),
  subtotal: money,
  tax: money,
  paymentMethod: z.string(),
  gstin: z.string().trim().max(20),
  items: z.array(z.object({ id: z.string().optional(), name: z.string().trim(), quantity: z.string(), unitPrice: money, amount: money })),
  notes: z.string().max(2000),
  tags: z.array(z.string()),
})
export type ReceiptFormValues = z.infer<typeof receiptSchema>

const fromPaise = (n?: number) => (n === undefined || n === null ? '' : (n / 100).toFixed(2))
const toPaise = (s: string) => (s === '' ? undefined : Math.round(parseFloat(s) * 100))

export function receiptToForm(r: Receipt): ReceiptFormValues {
  return {
    merchant: r.merchant ?? '',
    date: r.date ?? '',
    time: r.time ?? '',
    categoryId: r.categoryId ?? '',
    currency: r.currency ?? 'INR',
    total: fromPaise(r.total),
    subtotal: fromPaise(r.subtotal),
    tax: fromPaise(r.tax),
    paymentMethod: r.paymentMethod ?? '',
    gstin: r.gstin ?? '',
    items: (r.items ?? []).map((it) => ({ id: it.id, name: it.name, quantity: it.quantity?.toString() ?? '', unitPrice: fromPaise(it.unitPrice), amount: fromPaise(it.amount) })),
    notes: r.notes ?? '',
    tags: r.tags ?? [],
  }
}

export function formToPatch(v: ReceiptFormValues): ReceiptPatch {
  return {
    merchant: v.merchant,
    date: v.date,
    time: v.time,
    categoryId: v.categoryId || undefined,
    currency: v.currency.toUpperCase(),
    total: toPaise(v.total),
    subtotal: toPaise(v.subtotal),
    tax: toPaise(v.tax),
    paymentMethod: v.paymentMethod,
    gstin: v.gstin,
    items: v.items
      .filter((it) => it.name.trim())
      .map((it) => ({
        id: it.id,
        name: it.name.trim(),
        quantity: it.quantity ? parseFloat(it.quantity) : undefined,
        unitPrice: toPaise(it.unitPrice),
        amount: toPaise(it.amount) ?? 0,
      })),
    notes: v.notes,
    tags: v.tags,
  }
}

