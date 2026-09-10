import { useState, type ReactNode } from 'react'
import { Controller, useFieldArray, useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Plus, Trash2, X } from 'lucide-react'
import { Field, Input, Select, inputClass } from '@/components/ui/Field'
import { Alert } from '@/components/ui/Alert'
import type { Category } from '@/features/categories/types'
import { paymentMethods, type Receipt, type ReceiptPatch } from './types'
import { receiptSchema, receiptToForm, formToPatch, type ReceiptFormValues } from './receiptForm'

export function ReceiptForm({ receipt, categories, onSubmit, submitting, error, actions, compact }: {
  receipt: Receipt
  categories: Category[]
  onSubmit: (patch: ReceiptPatch) => void
  submitting?: boolean
  error?: string
  actions: (submit: () => void) => ReactNode
  compact?: boolean
}) {
  const { register, control, handleSubmit, formState: { errors } } = useForm<ReceiptFormValues>({
    resolver: zodResolver(receiptSchema),
    defaultValues: receiptToForm(receipt),
  })
  const items = useFieldArray({ control, name: 'items' })
  const submit = handleSubmit((v) => onSubmit(formToPatch(v)))
  const symbol = receipt.currency === 'INR' ? '₹' : receipt.currency

  return (
    <form onSubmit={(e) => { e.preventDefault(); submit() }} className="grid gap-4" noValidate>
      {error && <Alert>{error}</Alert>}
      <div className={compact ? 'grid gap-3' : 'grid gap-3 sm:grid-cols-2'}>
        <Field label="Merchant" error={errors.merchant?.message}>
          <Input invalid={!!errors.merchant} {...register('merchant')} />
        </Field>
        <Field label="Category">
          <Select {...register('categoryId')}>
            <option value="">Uncategorized</option>
            {categories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
          </Select>
        </Field>
        <Field label="Date" error={errors.date?.message}>
          <Input type="date" invalid={!!errors.date} {...register('date')} />
        </Field>
        <Field label="Time">
          <Input type="time" {...register('time')} />
        </Field>
        <Field label={`Total amount (${symbol})`} error={errors.total?.message}>
          <Input inputMode="decimal" placeholder="0.00" invalid={!!errors.total} {...register('total')} />
        </Field>
        <Field label={`Tax (${symbol})`} error={errors.tax?.message}>
          <Input inputMode="decimal" placeholder="0.00" invalid={!!errors.tax} {...register('tax')} />
        </Field>
        <Field label="Payment method">
          <Select {...register('paymentMethod')}>
            <option value="">Unknown</option>
            {paymentMethods.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
          </Select>
        </Field>
        <Field label="GSTIN">
          <Input placeholder="Optional" {...register('gstin')} />
        </Field>
      </div>

      <div>
        <div className="mb-2 flex items-center justify-between">
          <span className="text-sm font-medium">Items ({items.fields.length})</span>
          <button type="button" onClick={() => items.append({ name: '', quantity: '', unitPrice: '', amount: '' })} className="flex items-center gap-1 text-xs font-medium text-brand-700 hover:underline">
            <Plus className="size-3.5" /> Add Item
          </button>
        </div>
        <div className="grid gap-2">
          {items.fields.map((f, i) => (
            <div key={f.id} className="grid grid-cols-[1fr_64px_96px_32px] items-center gap-2">
              <Input placeholder="Item name" aria-label="Item name" {...register(`items.${i}.name`)} />
              <Input placeholder="Qty" aria-label="Quantity" inputMode="decimal" {...register(`items.${i}.quantity`)} />
              <Input placeholder="Amount" aria-label="Amount" inputMode="decimal" invalid={!!errors.items?.[i]?.amount} {...register(`items.${i}.amount`)} />
              <button type="button" aria-label="Remove item" onClick={() => items.remove(i)} className="text-muted grid size-8 place-items-center rounded-md hover:bg-red-50 hover:text-red-600">
                <Trash2 className="size-4" />
              </button>
            </div>
          ))}
          {items.fields.length === 0 && <p className="text-muted text-xs">No items. Add lines if you want itemized tracking.</p>}
        </div>
      </div>

      <Field label="Notes">
        <textarea rows={2} className={inputClass + ' h-auto py-2'} placeholder="Optional" {...register('notes')} />
      </Field>

      <Controller control={control} name="tags" render={({ field }) => <TagInput value={field.value} onChange={field.onChange} />} />

      <div className="flex flex-wrap justify-end gap-2">{actions(submit)}</div>
      {submitting && <p className="text-muted text-right text-xs">Saving…</p>}
    </form>
  )
}

export function TagInput({ value, onChange }: { value: string[]; onChange: (v: string[]) => void }) {
  const [draft, setDraft] = useState('')
  const add = () => {
    const t = draft.trim()
    if (t && !value.some((v) => v.toLowerCase() === t.toLowerCase())) onChange([...value, t])
    setDraft('')
  }
  return (
    <div>
      <span className="mb-1 block text-sm font-medium">Tags</span>
      <div className="flex flex-wrap items-center gap-1.5">
        {value.map((t) => (
          <span key={t} className="inline-flex items-center gap-1 rounded-full bg-brand-50 px-2 py-0.5 text-xs font-medium text-brand-700 dark:bg-brand-900/30 dark:text-brand-300">
            {t}
            <button type="button" aria-label={`Remove ${t}`} onClick={() => onChange(value.filter((v) => v !== t))}><X className="size-3" /></button>
          </span>
        ))}
        <input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ',') { e.preventDefault(); add() } }}
          onBlur={add}
          placeholder="Add tag"
          className="h-7 min-w-24 flex-1 rounded-md border border-dashed border-[var(--border)] bg-transparent px-2 text-xs outline-none focus:border-brand-500"
        />
      </div>
    </div>
  )
}
