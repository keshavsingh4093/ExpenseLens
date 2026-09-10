import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useParams } from 'react-router'
import { Pencil, Trash2, AlertTriangle, RefreshCw, Plus } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { PageHeader } from '@/components/ui/PageHeader'
import { Alert } from '@/components/ui/Alert'
import { Skeleton } from '@/components/ui/Skeleton'
import { ConfirmDialog } from '@/components/ui/Modal'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { CategoryBadge } from '@/features/categories/CategoryIcon'
import { useCategories } from '@/features/categories/useCategories'
import { ApiRequestError } from '@/lib/api/client'
import { formatMoney } from '@/lib/format/money'
import { formatDateTime } from '@/lib/format/date'
import { receiptsApi, receiptKeys } from './api'
import { ImageCarousel } from './ImageCarousel'
import { ReceiptForm, TagInput } from './ReceiptForm'
import { paymentMethods, type ReceiptPatch } from './types'

export function ReceiptDetailPage() {
  const { id = '' } = useParams()
  const nav = useNavigate()
  const qc = useQueryClient()
  const { categories, byId } = useCategories()
  const [editing, setEditing] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  const q = useQuery({
    queryKey: receiptKeys.detail(id),
    queryFn: () => receiptsApi.get(id),
    refetchInterval: (query) => (query.state.data?.status === 'processing' ? 1000 : false),
  })
  const invalidate = () => {
    qc.invalidateQueries({ queryKey: receiptKeys.all })
    qc.invalidateQueries({ queryKey: ['analytics'] })
  }
  const update = useMutation({
    mutationFn: (patch: ReceiptPatch) => receiptsApi.update(id, patch),
    onSuccess: (rc) => { qc.setQueryData(receiptKeys.detail(id), rc); invalidate(); setEditing(false) },
  })
  const remove = useMutation({
    mutationFn: () => receiptsApi.remove(id),
    onSuccess: () => { invalidate(); nav('/receipts', { replace: true }) },
  })
  const reextract = useMutation({
    mutationFn: () => receiptsApi.reextract(id),
    onSuccess: (rc) => { qc.setQueryData(receiptKeys.detail(id), rc); nav(`/receipts/${id}/review`) },
  })
  const addImages = useMutation({
    mutationFn: (files: File[]) => receiptsApi.addImages(id, files),
    onSuccess: (rc) => qc.setQueryData(receiptKeys.detail(id), rc),
  })
  const removeImage = useMutation({
    mutationFn: (imageId: string) => receiptsApi.removeImage(id, imageId),
    onSuccess: (rc) => qc.setQueryData(receiptKeys.detail(id), rc),
  })

  if (q.isLoading) return <div className="mx-auto grid max-w-4xl gap-4"><Skeleton className="h-8 w-48" /><Skeleton className="h-72" /></div>
  if (q.isError || !q.data) return <Alert>Could not load this receipt.</Alert>
  const rc = q.data
  const cat = rc.categoryId ? byId.get(rc.categoryId) : undefined
  const pm = paymentMethods.find(([v]) => v === rc.paymentMethod)?.[1]
  const itemsTotal = rc.items.reduce((s, it) => s + it.amount, 0)

  return (
    <div className="mx-auto max-w-4xl">
      <PageHeader
        title="Receipt Details"
        back="/receipts"
        actions={
          <>
            {!editing && <Button variant="secondary" size="sm" onClick={() => setEditing(true)}><Pencil className="size-4" /> Edit</Button>}
            <Button variant="danger" size="sm" onClick={() => setDeleteOpen(true)}><Trash2 className="size-4" /> Delete</Button>
          </>
        }
      />
      {rc.status !== 'confirmed' && (
        <div className="mb-4">
          <Alert kind={rc.status === 'failed' ? 'error' : 'info'}>
            <span className="flex flex-wrap items-center gap-2">
              <AlertTriangle className="size-4" />
              {rc.status === 'processing' ? 'Extraction is still running.' : rc.status === 'failed' ? 'Extraction failed for this receipt.' : 'This receipt has not been reviewed yet.'}
              {rc.status !== 'processing' && <Link to={`/receipts/${id}/review`} className="font-medium underline">Review now</Link>}
            </span>
          </Alert>
        </div>
      )}

      <div className="grid gap-5 md:grid-cols-[minmax(0,5fr)_minmax(0,7fr)]">
        <div>
          <ImageCarousel images={rc.images} onRemove={editing && rc.images.length > 1 ? (imgId) => removeImage.mutate(imgId) : undefined} />
          {editing && (
            <label className="mt-2 inline-flex cursor-pointer items-center gap-1 text-xs text-brand-700 hover:underline">
              <Plus className="size-3.5" /> Add page
              <input type="file" accept="image/*,application/pdf" multiple hidden onChange={(e) => { const f = Array.from(e.target.files ?? []); if (f.length) addImages.mutate(f); e.target.value = '' }} />
            </label>
          )}
          <button type="button" onClick={() => reextract.mutate()} disabled={reextract.isPending} className="text-muted mt-2 flex items-center gap-1 text-xs hover:text-brand-700">
            <RefreshCw className="size-3.5" /> Re-run extraction
          </button>
        </div>

        <div className="card p-4 md:p-5">
          {editing ? (
            <ReceiptForm
              receipt={rc}
              categories={categories}
              onSubmit={(patch) => update.mutate(patch)}
              submitting={update.isPending}
              error={update.error instanceof ApiRequestError ? update.error.message : undefined}
              actions={(submit) => (
                <>
                  <Button type="button" variant="secondary" onClick={() => setEditing(false)}>Cancel</Button>
                  <Button type="button" onClick={submit} loading={update.isPending}>Save</Button>
                </>
              )}
            />
          ) : (
            <>
              <dl className="grid gap-y-2 text-sm">
                <Row label="Merchant"><span className="font-medium">{rc.merchant || '—'}</span> <StatusBadge status={rc.status} className="ml-2" /></Row>
                <Row label="Date">{formatDateTime(rc.date, rc.time) || '—'}</Row>
                <Row label="Category"><CategoryBadge category={cat} /></Row>
                <Row label="Total Amount"><span className="text-base font-semibold">{formatMoney(rc.total, rc.currency)}</span></Row>
                {rc.subtotal !== undefined && <Row label="Subtotal">{formatMoney(rc.subtotal, rc.currency)}</Row>}
                {rc.tax !== undefined && <Row label="Tax">{formatMoney(rc.tax, rc.currency)}</Row>}
                {rc.taxBreakdown?.map((t) => <Row key={t.label} label={t.label} muted>{formatMoney(t.amount, rc.currency)}</Row>)}
                <Row label="Payment Method">{pm ?? '—'}</Row>
                {rc.gstin && <Row label="GSTIN"><span className="font-mono text-xs">{rc.gstin}</span></Row>}
              </dl>

              <h3 className="mt-5 mb-2 text-sm font-semibold">Items ({rc.items.length})</h3>
              {rc.items.length ? (
                <ul className="divide-y divide-[var(--border)] rounded-lg border border-[var(--border)]">
                  {rc.items.map((it) => (
                    <li key={it.id ?? it.name} className="flex items-center justify-between px-3 py-2 text-sm">
                      <span>{it.name}{it.quantity ? <span className="text-muted"> × {it.quantity}</span> : null}</span>
                      <span className="tabular-nums">{formatMoney(it.amount, rc.currency)}</span>
                    </li>
                  ))}
                  {itemsTotal !== rc.total && (
                    <li className="text-muted flex justify-between px-3 py-2 text-xs">
                      <span>Items subtotal</span><span className="tabular-nums">{formatMoney(itemsTotal, rc.currency)}</span>
                    </li>
                  )}
                </ul>
              ) : (
                <p className="text-muted text-sm">No items recorded.</p>
              )}

              <h3 className="mt-5 mb-1 text-sm font-semibold">Notes</h3>
              <p className="text-sm whitespace-pre-wrap">{rc.notes || <span className="text-muted">No notes.</span>}</p>

              <div className="mt-5">
                <TagInput value={rc.tags} onChange={(tags) => update.mutate({ tags })} />
              </div>

              {rc.extracted?.warrantyCandidates?.length ? (
                <div className="mt-5">
                  <Alert kind="info">
                    Possible warranty items: {rc.extracted.warrantyCandidates.map((w) => w.itemName).join(', ')}. Track them under Warranty.
                  </Alert>
                </div>
              ) : null}
            </>
          )}
        </div>
      </div>

      <ConfirmDialog
        open={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        onConfirm={() => remove.mutate()}
        loading={remove.isPending}
        title="Delete receipt?"
        message="This permanently removes the receipt and its images."
      />
    </div>
  )
}

function Row({ label, children, muted }: { label: string; children: React.ReactNode; muted?: boolean }) {
  return (
    <div className="grid grid-cols-[130px_1fr] items-center gap-2">
      <dt className={muted ? 'text-muted pl-3 text-xs' : 'text-muted'}>{label}</dt>
      <dd className="min-w-0">{children}</dd>
    </div>
  )
}
