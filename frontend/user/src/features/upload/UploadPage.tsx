import { useCallback, useRef, useState } from 'react'
import { useDropzone } from 'react-dropzone'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from 'react-router'
import { CloudUpload, Camera, FileText, Image as ImageIcon, CheckCircle2, AlertCircle, Copy, Loader2 } from 'lucide-react'
import { clsx } from 'clsx'
import { Button } from '@common/ui/Button'
import { PageHeader } from '@common/ui/PageHeader'
import { StatusBadge } from '@common/ui/StatusBadge'
import { Skeleton } from '@common/ui/Skeleton'
import { receiptsApi, receiptKeys } from '@/features/receipts/api'
import type { UploadResult } from '@/features/receipts/types'
import { formatBytes, formatDate } from '@common/format/date'
import { ApiRequestError } from '@common/api/client'

type Row = { name: string; size: number; state: 'uploading' | 'done' | 'duplicate' | 'error'; id?: string; error?: string }

export function UploadPage() {
  const nav = useNavigate()
  const qc = useQueryClient()
  const [rows, setRows] = useState<Row[]>([])
  const [group, setGroup] = useState(false)
  const cameraRef = useRef<HTMLInputElement>(null)
  const pdfRef = useRef<HTMLInputElement>(null)

  const upload = useMutation({
    mutationFn: ({ files, group }: { files: File[]; group: boolean }) => receiptsApi.upload(files, group),
    onSuccess: (data, vars) => {
      qc.invalidateQueries({ queryKey: receiptKeys.all })
      const results = data.receipts
      setRows((prev) =>
        prev.map((r) => {
          const hit: UploadResult | undefined = vars.group ? results[0] : results.find((x) => x.fileName === r.name)
          if (!hit || r.state !== 'uploading') return r
          return { ...r, state: hit.duplicate ? 'duplicate' : 'done', id: hit.id }
        }),
      )
      if (results.length === 1 && !results[0].duplicate) nav(`/receipts/${results[0].id}/review`)
    },
    onError: (e, vars) => {
      const msg = e instanceof ApiRequestError ? e.message : 'Upload failed'
      setRows((prev) => prev.map((r) => (vars.files.some((f) => f.name === r.name) && r.state === 'uploading' ? { ...r, state: 'error', error: msg } : r)))
    },
  })

  const start = useCallback(
    (files: File[]) => {
      if (!files.length) return
      setRows((prev) => [...files.map((f) => ({ name: f.name, size: f.size, state: 'uploading' as const })), ...prev])
      upload.mutate({ files, group: group && files.length > 1 })
    },
    [upload, group],
  )

  const { getRootProps, getInputProps, isDragActive, open } = useDropzone({
    onDrop: start,
    noClick: true,
    multiple: true,
    accept: { 'image/jpeg': [], 'image/png': [], 'image/webp': [], 'image/heic': [], 'application/pdf': [] },
    maxSize: 20 * 1024 * 1024,
  })

  const recent = useQuery({ queryKey: receiptKeys.recent, queryFn: () => receiptsApi.recent(5) })

  return (
    <div className="mx-auto max-w-2xl">
      <PageHeader title="Upload Receipt" back="/" />

      <div
        {...getRootProps()}
        className={clsx(
          'grid place-items-center rounded-card border-2 border-dashed px-6 py-10 text-center transition-colors',
          isDragActive ? 'border-brand-500 bg-brand-50 dark:bg-brand-900/20' : 'border-[var(--border)] bg-[var(--surface)]',
        )}
      >
        <input {...getInputProps()} />
        <span className="grid size-14 place-items-center rounded-full bg-brand-100 text-brand-600 dark:bg-brand-900/40">
          <CloudUpload className="size-7" />
        </span>
        <p className="mt-3 text-sm font-medium">Drag &amp; drop your receipt here</p>
        <p className="text-muted my-2 text-xs">or</p>
        <Button onClick={open}>Browse Files</Button>
        <p className="text-muted mt-3 text-xs">JPEG, PNG, WebP, HEIC or PDF · up to 20 MB each</p>
      </div>

      <label className="text-muted mt-3 flex items-center gap-2 text-xs">
        <input type="checkbox" checked={group} onChange={(e) => setGroup(e.target.checked)} className="accent-brand-600" />
        Combine multiple files into one receipt (multi-page)
      </label>

      <div className="text-muted my-4 text-center text-xs uppercase">or</div>
      <div className="grid grid-cols-2 gap-3">
        <button type="button" onClick={() => cameraRef.current?.click()} className="card flex flex-col items-center gap-2 py-5 text-sm font-medium hover:border-brand-400">
          <Camera className="size-6 text-brand-600" /> Take Photo
        </button>
        <button type="button" onClick={() => pdfRef.current?.click()} className="card flex flex-col items-center gap-2 py-5 text-sm font-medium hover:border-brand-400">
          <FileText className="size-6 text-brand-600" /> Upload PDF
        </button>
        <input ref={cameraRef} type="file" accept="image/*" capture="environment" hidden onChange={(e) => { start(Array.from(e.target.files ?? [])); e.target.value = '' }} />
        <input ref={pdfRef} type="file" accept="application/pdf" multiple hidden onChange={(e) => { start(Array.from(e.target.files ?? [])); e.target.value = '' }} />
      </div>

      {rows.length > 0 && (
        <section className="mt-6">
          <h2 className="mb-2 text-sm font-semibold">This session</h2>
          <ul className="grid gap-2">
            {rows.map((r, i) => (
              <li key={`${r.name}-${i}`} className="card flex items-center gap-3 px-3 py-2 text-sm">
                <FileIcon name={r.name} />
                <div className="min-w-0 flex-1">
                  <div className="truncate font-medium">{r.name}</div>
                  <div className="text-muted text-xs">{formatBytes(r.size)}{r.error ? ` · ${r.error}` : ''}</div>
                </div>
                {r.state === 'uploading' && <Loader2 className="size-4 animate-spin text-brand-600" />}
                {r.state === 'done' && <Link to={`/receipts/${r.id}/review`} className="flex items-center gap-1 text-xs text-brand-700"><CheckCircle2 className="size-4" /> Review</Link>}
                {r.state === 'duplicate' && <Link to={`/receipts/${r.id}`} className="flex items-center gap-1 text-xs text-amber-700"><Copy className="size-4" /> Already uploaded</Link>}
                {r.state === 'error' && <AlertCircle className="size-4 text-red-600" />}
              </li>
            ))}
          </ul>
        </section>
      )}

      <section className="mt-6">
        <h2 className="mb-2 text-sm font-semibold">Recent Uploads</h2>
        {recent.isLoading ? (
          <div className="grid gap-2"><Skeleton className="h-14" /><Skeleton className="h-14" /></div>
        ) : recent.data?.receipts.length ? (
          <ul className="grid gap-2">
            {recent.data.receipts.map((rc) => {
              const img = rc.images[0]
              const to = rc.status === 'confirmed' ? `/receipts/${rc.id}` : `/receipts/${rc.id}/review`
              return (
                <li key={rc.id}>
                  <Link to={to} className="card flex items-center gap-3 px-3 py-2 text-sm hover:border-brand-400">
                    <FileIcon name={img?.fileName ?? 'receipt'} />
                    <div className="min-w-0 flex-1">
                      <div className="truncate font-medium">{rc.merchant || img?.fileName || 'Receipt'}</div>
                      <div className="text-muted text-xs">{formatDate(rc.createdAt)}{img ? ` · ${formatBytes(img.size)}` : ''}</div>
                    </div>
                    <StatusBadge status={rc.status} />
                  </Link>
                </li>
              )
            })}
          </ul>
        ) : (
          <p className="text-muted text-sm">Nothing uploaded yet.</p>
        )}
      </section>
    </div>
  )
}

function FileIcon({ name }: { name: string }) {
  const pdf = name.toLowerCase().endsWith('.pdf')
  return (
    <span className={clsx('grid size-9 shrink-0 place-items-center rounded-lg', pdf ? 'bg-red-50 text-red-600 dark:bg-red-950/40' : 'bg-blue-50 text-blue-600 dark:bg-blue-950/40')}>
      {pdf ? <FileText className="size-4" /> : <ImageIcon className="size-4" />}
    </span>
  )
}
