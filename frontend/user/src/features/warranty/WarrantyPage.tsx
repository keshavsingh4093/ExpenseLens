import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router'
import { Pencil, Plus, Receipt as ReceiptIcon, ShieldAlert, ShieldCheck, ShieldX, Trash2 } from 'lucide-react'
import { clsx } from 'clsx'
import { Button } from '@common/ui/Button'
import { Chip } from '@common/ui/Chip'
import { EmptyState } from '@common/ui/EmptyState'
import { Skeleton } from '@common/ui/Skeleton'
import { Modal, ConfirmDialog } from '@common/ui/Modal'
import { Field, Input, Select, inputClass } from '@common/ui/Field'
import { Alert } from '@common/ui/Alert'
import { formatDate } from '@common/format/date'
import { ApiRequestError } from '@common/api/client'
import { receiptsApi, receiptKeys } from '@/features/receipts/api'
import { warrantyApi, warrantyKeys, type Warranty, type WarrantyInput, type WarrantyStatus } from './api'

const tabs: [WarrantyStatus, string][] = [['active', 'Active'], ['expiring', 'Expiring soon'], ['expired', 'Expired']]

const tone: Record<WarrantyStatus, { icon: typeof ShieldCheck; cls: string; label: (d: number) => string }> = {
  active: { icon: ShieldCheck, cls: 'bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300', label: (d) => `${d} days left` },
  expiring: { icon: ShieldAlert, cls: 'bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300', label: (d) => (d <= 0 ? 'Expires today' : d === 1 ? 'Expires tomorrow' : `${d} days left`) },
  expired: { icon: ShieldX, cls: 'bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300', label: (d) => `Expired ${-d} day${d === -1 ? '' : 's'} ago` },
}

export function WarrantyPage() {
  const qc = useQueryClient()
  const [tab, setTab] = useState<WarrantyStatus>('active')
  const q = useQuery({ queryKey: warrantyKeys.all, queryFn: () => warrantyApi.list() })
  const [editing, setEditing] = useState<Warranty | 'new' | null>(null)
  const [deleting, setDeleting] = useState<Warranty | null>(null)
  const del = useMutation({ mutationFn: (id: string) => warrantyApi.remove(id), onSuccess: () => { setDeleting(null); qc.invalidateQueries({ queryKey: warrantyKeys.all }) } })

  const rows = (q.data?.warranties ?? []).filter((w) => w.status === tab)
  const counts = q.data?.counts

  return (
    <div className="mx-auto max-w-3xl">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Warranty</h1>
          <p className="text-muted text-sm">Track product warranties and get reminded before they expire.</p>
        </div>
        <Button size="sm" onClick={() => setEditing('new')}><Plus className="size-4" /> Add warranty</Button>
      </div>

      <div className="mb-4 flex gap-2 overflow-x-auto pb-1" role="tablist">
        {tabs.map(([v, l]) => (
          <Chip key={v} role="tab" aria-selected={tab === v} active={tab === v} onClick={() => setTab(v)}>
            {l}{counts ? ` · ${counts[v]}` : ''}
          </Chip>
        ))}
      </div>

      {q.isLoading ? (
        <div className="grid gap-3"><Skeleton className="h-24" /><Skeleton className="h-24" /></div>
      ) : rows.length === 0 ? (
        <div className="card">
          <EmptyState icon={tone[tab].icon} title={tab === 'active' ? 'No active warranties' : tab === 'expiring' ? 'Nothing expiring in the next 30 days' : 'No expired warranties'} description={tab === 'active' ? 'Add a warranty for a product you bought, optionally linked to its receipt.' : undefined} action={tab === 'active' ? <Button size="sm" onClick={() => setEditing('new')}>Add warranty</Button> : undefined} />
        </div>
      ) : (
        <ul className="grid gap-3">
          {rows.map((w) => {
            const t = tone[w.status]
            return (
              <li key={w.id} className="card flex gap-3 p-4">
                {w.imageUrl ? (
                  <img src={w.imageUrl} alt="" className="size-14 shrink-0 rounded-lg object-cover" />
                ) : (
                  <span className={clsx('grid size-14 shrink-0 place-items-center rounded-lg', t.cls)}><t.icon className="size-6" /></span>
                )}
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="truncate text-sm font-semibold">{w.productName}</span>
                    <span className={clsx('rounded-full px-2 py-0.5 text-xs font-medium', t.cls)}>{t.label(w.daysLeft)}</span>
                  </div>
                  <div className="text-muted mt-0.5 text-xs">Bought {formatDate(w.purchaseDate)} · Expires {formatDate(w.expiryDate)}{w.durationMonths ? ` · ${w.durationMonths} months` : ''}</div>
                  {w.notes && <div className="text-muted mt-1 line-clamp-2 text-xs">{w.notes}</div>}
                  {w.receiptId && (
                    <Link to={`/receipts/${w.receiptId}`} className="mt-1 inline-flex items-center gap-1 text-xs font-medium text-brand-700 hover:underline"><ReceiptIcon className="size-3.5" /> View receipt</Link>
                  )}
                </div>
                <div className="flex shrink-0 flex-col gap-1">
                  <button type="button" aria-label={`Edit ${w.productName}`} onClick={() => setEditing(w)} className="text-muted rounded-md p-1.5 hover:bg-brand-50 hover:text-brand-700 dark:hover:bg-brand-900/30"><Pencil className="size-4" /></button>
                  <button type="button" aria-label={`Delete ${w.productName}`} onClick={() => setDeleting(w)} className="text-muted rounded-md p-1.5 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950/40"><Trash2 className="size-4" /></button>
                </div>
              </li>
            )
          })}
        </ul>
      )}

      {editing && <WarrantyModal warranty={editing === 'new' ? null : editing} onClose={() => setEditing(null)} onSaved={(w) => { setEditing(null); setTab(w.status); qc.invalidateQueries({ queryKey: warrantyKeys.all }) }} />}
      <ConfirmDialog open={!!deleting} onClose={() => setDeleting(null)} onConfirm={() => deleting && del.mutate(deleting.id)} loading={del.isPending} title="Delete warranty" message={`Remove the warranty for ${deleting?.productName ?? ''}?`} />
    </div>
  )
}

function WarrantyModal({ warranty, onClose, onSaved }: { warranty: Warranty | null; onClose: () => void; onSaved: (w: Warranty) => void }) {
  const [productName, setProductName] = useState(warranty?.productName ?? '')
  const [purchaseDate, setPurchaseDate] = useState(warranty?.purchaseDate ?? '')
  const [mode, setMode] = useState<'duration' | 'date'>(warranty && !warranty.durationMonths ? 'date' : 'duration')
  const [durationMonths, setDurationMonths] = useState(String(warranty?.durationMonths ?? 12))
  const [expiryDate, setExpiryDate] = useState(warranty?.expiryDate ?? '')
  const [receiptId, setReceiptId] = useState(warranty?.receiptId ?? '')
  const [itemId, setItemId] = useState(warranty?.itemId ?? '')
  const [notes, setNotes] = useState(warranty?.notes ?? '')
  const [error, setError] = useState('')
  const [fields, setFields] = useState<Record<string, string>>({})

  const receipts = useQuery({ queryKey: receiptKeys.list({ page: 1, pageSize: 50, sort: 'date_desc' }), queryFn: () => receiptsApi.list({ page: 1, pageSize: 50, sort: 'date_desc' }) })
  const receipt = useQuery({ queryKey: receiptKeys.detail(receiptId), queryFn: () => receiptsApi.get(receiptId), enabled: !!receiptId })
  // Purchase date defaults to the linked receipt's date until the user types one.
  const effectivePurchaseDate = purchaseDate || (!warranty && receipt.data?.date) || ''

  const save = useMutation({
    mutationFn: () => {
      const body: WarrantyInput = { productName, purchaseDate: effectivePurchaseDate, notes, receiptId, itemId: receiptId ? itemId : '' }
      if (mode === 'duration') {
        body.durationMonths = Number(durationMonths) || 0
      } else {
        body.expiryDate = expiryDate
        body.durationMonths = 0
      }
      return warranty ? warrantyApi.update(warranty.id, body) : warrantyApi.create(body)
    },
    onSuccess: onSaved,
    onError: (e) => {
      if (e instanceof ApiRequestError) { setFields(e.fields ?? {}); setError(e.fields ? '' : e.message) } else setError((e as Error).message)
    },
  })

  return (
    <Modal open onClose={onClose} title={warranty ? 'Edit warranty' : 'Add warranty'} footer={
      <>
        <Button variant="secondary" onClick={onClose}>Cancel</Button>
        <Button onClick={() => save.mutate()} loading={save.isPending}>{warranty ? 'Save' : 'Add'}</Button>
      </>
    }>
      <form className="grid gap-3" onSubmit={(e) => { e.preventDefault(); save.mutate() }} noValidate>
        {error && <Alert>{error}</Alert>}
        <Field label="Product" error={fields.productName}>
          <Input name="productName" value={productName} onChange={(e) => setProductName(e.target.value)} placeholder="e.g. Samsung 43&quot; TV" invalid={!!fields.productName} autoFocus />
        </Field>
        <Field label="Receipt (optional)" hint="Links the warranty to a receipt and uses its photo.">
          <Select name="receiptId" value={receiptId} onChange={(e) => { setReceiptId(e.target.value); setItemId('') }}>
            <option value="">No receipt</option>
            {receipts.data?.receipts.map((r) => <option key={r.id} value={r.id}>{r.merchant || 'Unknown merchant'} · {formatDate(r.date || r.createdAt)}</option>)}
          </Select>
        </Field>
        {receiptId && (receipt.data?.items.length ?? 0) > 0 && (
          <Field label="Item">
            <Select name="itemId" value={itemId} onChange={(e) => { setItemId(e.target.value); const it = receipt.data?.items.find((i) => i.id === e.target.value); if (it && !productName) setProductName(it.name) }}>
              <option value="">Whole receipt</option>
              {receipt.data!.items.map((it) => <option key={it.id} value={it.id}>{it.name}</option>)}
            </Select>
          </Field>
        )}
        <Field label="Purchase date" error={fields.purchaseDate}>
          <Input name="purchaseDate" type="date" value={effectivePurchaseDate} onChange={(e) => setPurchaseDate(e.target.value)} invalid={!!fields.purchaseDate} />
        </Field>
        <div className="grid grid-cols-[auto_1fr] items-end gap-2">
          <Field label="Warranty">
            <Select value={mode} onChange={(e) => setMode(e.target.value as 'duration' | 'date')} aria-label="Warranty length mode" className="w-auto!">
              <option value="duration">Duration</option>
              <option value="date">Expiry date</option>
            </Select>
          </Field>
          {mode === 'duration' ? (
            <Field label="Months" error={fields.expiryDate}>
              <Input name="durationMonths" inputMode="numeric" value={durationMonths} onChange={(e) => setDurationMonths(e.target.value)} invalid={!!fields.expiryDate} />
            </Field>
          ) : (
            <Field label="Expires on" error={fields.expiryDate}>
              <Input name="expiryDate" type="date" value={expiryDate} onChange={(e) => setExpiryDate(e.target.value)} invalid={!!fields.expiryDate} />
            </Field>
          )}
        </div>
        <Field label="Notes">
          <textarea name="notes" value={notes} onChange={(e) => setNotes(e.target.value)} rows={2} className={clsx(inputClass, 'h-auto py-2')} placeholder="Serial number, dealer, extended cover…" />
        </Field>
        <button type="submit" hidden />
      </form>
    </Modal>
  )
}
