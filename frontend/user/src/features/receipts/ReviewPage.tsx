import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useParams } from 'react-router'
import { Loader2, RefreshCw, Sparkles } from 'lucide-react'
import { Button } from '@common/ui/Button'
import { PageHeader } from '@common/ui/PageHeader'
import { Alert } from '@common/ui/Alert'
import { Skeleton } from '@common/ui/Skeleton'
import { ConfirmDialog } from '@common/ui/Modal'
import { useCategories } from '@/features/categories/useCategories'
import { ApiRequestError } from '@common/api/client'
import { receiptsApi, receiptKeys } from './api'
import { ImageCarousel } from './ImageCarousel'
import { ReceiptForm } from './ReceiptForm'
import type { ReceiptPatch } from './types'

export function ReviewPage() {
  const { id = '' } = useParams()
  const nav = useNavigate()
  const qc = useQueryClient()
  const { categories } = useCategories()
  const [discardOpen, setDiscardOpen] = useState(false)

  const q = useQuery({
    queryKey: receiptKeys.detail(id),
    queryFn: () => receiptsApi.get(id),
    refetchInterval: (query) => (query.state.data?.status === 'processing' ? 1000 : false),
  })

  const confirm = useMutation({
    mutationFn: (patch: ReceiptPatch) => receiptsApi.confirm(id, patch),
    onSuccess: (rc) => {
      qc.setQueryData(receiptKeys.detail(id), rc)
      qc.invalidateQueries({ queryKey: receiptKeys.all })
      qc.invalidateQueries({ queryKey: ['analytics'] })
      nav(`/receipts/${id}`, { replace: true })
    },
  })
  const reextract = useMutation({
    mutationFn: () => receiptsApi.reextract(id),
    onSuccess: (rc) => qc.setQueryData(receiptKeys.detail(id), rc),
  })
  const discard = useMutation({
    mutationFn: () => receiptsApi.remove(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: receiptKeys.all })
      nav('/upload', { replace: true })
    },
  })

  if (q.isLoading) return <div className="mx-auto grid max-w-3xl gap-4"><Skeleton className="h-8 w-48" /><Skeleton className="h-72" /></div>
  if (q.isError || !q.data) return <Alert>Could not load this receipt.</Alert>
  const rc = q.data

  return (
    <div className="mx-auto max-w-4xl">
      <PageHeader title="Review Receipt" back="/upload" />
      <div className="grid gap-5 md:grid-cols-[minmax(0,5fr)_minmax(0,7fr)]">
        <div>
          <ImageCarousel images={rc.images} />
          {rc.extracted && rc.status !== 'processing' && !rc.extracted.error && (
            <p className="text-muted mt-2 flex items-center gap-1 text-xs">
              <Sparkles className="size-3.5 text-brand-600" />
              Extracted by {rc.extracted.model} · confidence {Math.round(rc.extracted.confidence * 100)}%
            </p>
          )}
        </div>
        <div className="card p-4 md:p-5">
          {rc.status === 'processing' ? (
            <div className="grid place-items-center gap-3 py-16 text-center">
              <Loader2 className="size-8 animate-spin text-brand-600" />
              <div className="font-medium">Extracting details…</div>
              <p className="text-muted text-sm">Reading merchant, date, items and totals from your receipt.</p>
            </div>
          ) : (
            <>
              <h2 className="mb-3 text-base font-semibold">Extracted Details</h2>
              {rc.status === 'failed' && (
                <div className="mb-4">
                  <Alert>
                    Automatic extraction failed{rc.extracted?.error ? `: ${rc.extracted.error}` : ''}. Fill in the details manually or retry.
                    <div className="mt-2">
                      <Button size="sm" variant="secondary" onClick={() => reextract.mutate()} loading={reextract.isPending}>
                        <RefreshCw className="size-3.5" /> Retry extraction
                      </Button>
                    </div>
                  </Alert>
                </div>
              )}
              <ReceiptForm
                key={rc.updatedAt}
                receipt={rc}
                categories={categories}
                onSubmit={(patch) => confirm.mutate(patch)}
                submitting={confirm.isPending}
                error={confirm.error instanceof ApiRequestError ? confirm.error.message : confirm.error ? 'Could not save' : undefined}
                actions={(submit) => (
                  <>
                    <Button type="button" variant="ghost" onClick={() => setDiscardOpen(true)}>Discard</Button>
                    <Button type="button" variant="secondary" onClick={() => nav('/upload')}>Cancel</Button>
                    <Button type="button" onClick={submit} loading={confirm.isPending}>Save Receipt</Button>
                  </>
                )}
              />
            </>
          )}
        </div>
      </div>
      <ConfirmDialog
        open={discardOpen}
        onClose={() => setDiscardOpen(false)}
        onConfirm={() => discard.mutate()}
        loading={discard.isPending}
        title="Discard receipt?"
        message="This deletes the uploaded file and extracted data."
        confirmLabel="Discard"
      />
    </div>
  )
}
