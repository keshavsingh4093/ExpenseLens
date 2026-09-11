import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, BarChart3, CircleCheck, Gauge, Repeat, RotateCw, ScanText, Table2, type LucideIcon } from 'lucide-react'
import { Bar, BarChart, CartesianGrid, Rectangle, ResponsiveContainer, Tooltip, XAxis, YAxis, type RectangleProps } from 'recharts'
import { PageHeader } from '@common/ui/PageHeader'
import { Button } from '@common/ui/Button'
import { Chip } from '@common/ui/Chip'
import { Skeleton } from '@common/ui/Skeleton'
import { EmptyState } from '@common/ui/EmptyState'
import { Alert } from '@common/ui/Alert'
import { ApiRequestError } from '@common/api/client'
import { extractionApi, extractionKeys, type ExtractionDay, type ExtractionRange, type FailedReceipt } from './api'

const ranges: { value: ExtractionRange; label: string }[] = [
  { value: '7d', label: '7 days' },
  { value: '30d', label: '30 days' },
  { value: '90d', label: '90 days' },
]
const FAILED_PAGE = 10

type DayRow = ExtractionDay & { label: string; succeeded: number }

const dayLabel = (iso: string) =>
  new Date(`${iso}T00:00:00Z`).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', timeZone: 'UTC' })
const when = (iso: string) => new Date(iso).toLocaleString('en-IN', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
const plural = (n: number, word: string) => `${n.toLocaleString('en-IN')} ${word}${n === 1 ? '' : 's'}`

export function ExtractionPage() {
  const [range, setRange] = useState<ExtractionRange>('30d')
  const stats = useQuery({ queryKey: extractionKeys.stats(range), queryFn: () => extractionApi.stats(range), refetchInterval: 30_000, placeholderData: (p) => p })
  const s = stats.data
  const t = s?.totals
  const days: DayRow[] = (s?.byDay ?? []).map((d) => ({ ...d, label: dayLabel(d.date), succeeded: Math.max(0, d.total - d.failed - d.processing) }))

  return (
    <div>
      <PageHeader
        title="Extraction"
        subtitle={s ? `Configured: ${s.extractor} · ${s.model}` : 'Receipt extraction volume and failures'}
        actions={
          <div className="flex gap-2" role="group" aria-label="Range">
            {ranges.map((r) => (
              <Chip key={r.value} active={range === r.value} aria-pressed={range === r.value} onClick={() => setRange(r.value)}>{r.label}</Chip>
            ))}
          </div>
        }
      />
      {stats.error && <Alert>Could not load extraction stats.</Alert>}
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {!t || !s ? (
          Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-28" />)
        ) : (
          <>
            <Tile icon={ScanText} label="Receipts processed" value={t.total.toLocaleString('en-IN')} sub={`${t.succeeded.toLocaleString('en-IN')} succeeded · ${t.processing.toLocaleString('en-IN')} in progress`} />
            <Tile icon={AlertTriangle} label="Failure rate" value={t.total ? `${s.failurePct.toFixed(1)}%` : '—'} sub={`${plural(t.failed, 'failed receipt')}`} />
            <Tile icon={Gauge} label="Average confidence" value={t.succeeded ? `${Math.round(t.avgConfidence * 100)}%` : '—'} sub="Successful extractions only" />
            <Tile icon={Repeat} label="Average attempts" value={t.total ? t.avgAttempts.toFixed(1) : '—'} sub="Model calls per receipt, incl. retries" />
          </>
        )}
      </div>

      <div className="mt-4 grid gap-4 lg:grid-cols-3">
        <VolumeCard days={days} loading={!s} />
        <section className="card p-4">
          <h2 className="text-base font-semibold">By model</h2>
          {!s ? (
            <Skeleton className="mt-3 h-24" />
          ) : s.byModel.length === 0 ? (
            <p className="text-muted mt-3 text-sm">No extractions in this range.</p>
          ) : (
            <div className="mt-2 overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="text-muted text-left text-xs">
                  <tr><th className="py-2 font-medium">Model</th><th className="py-2 text-right font-medium">Receipts</th><th className="py-2 text-right font-medium">Failed</th><th className="py-2 text-right font-medium">Confidence</th></tr>
                </thead>
                <tbody className="divide-y divide-[var(--border)]">
                  {s.byModel.map((m) => (
                    <tr key={m.model}>
                      <td className="py-2 font-mono text-xs break-all">{m.model || 'unknown'}</td>
                      <td className="py-2 text-right tabular-nums">{m.total.toLocaleString('en-IN')}</td>
                      <td className="py-2 text-right tabular-nums">{m.failed.toLocaleString('en-IN')}</td>
                      <td className="py-2 text-right tabular-nums">{m.total > m.failed ? `${Math.round(m.avgConfidence * 100)}%` : '—'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>

      <FailedList />
    </div>
  )
}

function Tile({ icon: Icon, label, value, sub }: { icon: LucideIcon; label: string; value: string; sub: string }) {
  return (
    <div className="card p-4">
      <div className="flex items-center gap-2">
        <span className="grid size-8 place-items-center rounded-lg bg-brand-50 text-brand-600 dark:bg-brand-900/30"><Icon className="size-4" /></span>
        <span className="text-muted text-xs">{label}</span>
      </div>
      <div className="mt-2 text-2xl font-semibold">{value}</div>
      <div className="text-muted mt-1 text-xs">{sub}</div>
    </div>
  )
}

function VolumeCard({ days, loading }: { days: DayRow[]; loading: boolean }) {
  const [asTable, setAsTable] = useState(false)
  const empty = days.every((d) => d.total === 0)
  return (
    <section className="card p-4 lg:col-span-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-base font-semibold">Receipts per day</h2>
        <Button variant="ghost" size="sm" aria-pressed={asTable} onClick={() => setAsTable((v) => !v)}>
          {asTable ? <><BarChart3 className="size-4" /> Chart</> : <><Table2 className="size-4" /> Table</>}
        </Button>
      </div>
      <div className="mt-2 flex flex-wrap gap-4 text-xs" aria-label="Legend">
        <span className="inline-flex items-center gap-1.5"><span className="size-2.5 rounded-sm" style={{ background: 'var(--viz-ok)' }} /><CircleCheck className="text-muted size-3.5" />Succeeded</span>
        <span className="inline-flex items-center gap-1.5"><span className="size-2.5 rounded-sm" style={{ background: 'var(--viz-failed)' }} /><AlertTriangle className="text-muted size-3.5" />Failed</span>
      </div>
      {loading ? (
        <Skeleton className="mt-3 h-56" />
      ) : empty ? (
        <p className="text-muted grid h-56 place-items-center text-sm">No receipts uploaded in this range.</p>
      ) : asTable ? (
        <div className="mt-3 max-h-56 overflow-auto">
          <table className="w-full text-sm">
            <thead className="text-muted sticky top-0 bg-[var(--surface)] text-left text-xs">
              <tr><th className="py-1.5 font-medium">Date</th><th className="py-1.5 text-right font-medium">Succeeded</th><th className="py-1.5 text-right font-medium">Failed</th><th className="py-1.5 text-right font-medium">In progress</th></tr>
            </thead>
            <tbody className="divide-y divide-[var(--border)]">
              {days.filter((d) => d.total > 0).reverse().map((d) => (
                <tr key={d.date}>
                  <td className="py-1.5">{d.label}</td>
                  <td className="py-1.5 text-right tabular-nums">{d.succeeded}</td>
                  <td className="py-1.5 text-right tabular-nums">{d.failed}</td>
                  <td className="py-1.5 text-right tabular-nums">{d.processing}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <VolumeChart days={days} />
      )}
    </section>
  )
}

/** Top of the visible stack gets the 4px rounded data-end; the baseline stays square. */
function segment(props: unknown, key: 'succeeded' | 'failed') {
  const p = props as RectangleProps & { payload: DayRow }
  const top = key === 'failed' || p.payload.failed === 0
  return <Rectangle {...p} radius={top ? [4, 4, 0, 0] : 0} />
}

function VolumeChart({ days }: { days: DayRow[] }) {
  const every = Math.max(1, Math.ceil(days.length / 8))
  return (
    <div className="mt-3 h-56 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={days} margin={{ top: 8, right: 4, left: 0, bottom: 0 }} barCategoryGap="20%">
          <CartesianGrid vertical={false} stroke="var(--border)" strokeWidth={1} />
          <XAxis dataKey="label" tickLine={false} axisLine={{ stroke: 'var(--border)' }} interval={every - 1} tick={{ fontSize: 11, fill: 'var(--text-muted)' }} dy={6} />
          <YAxis allowDecimals={false} tickLine={false} axisLine={false} width={32} tick={{ fontSize: 11, fill: 'var(--text-muted)' }} />
          <Tooltip
            cursor={{ fill: 'var(--border)', fillOpacity: 0.45 }}
            content={({ active, payload }) => {
              if (!active || !payload?.length) return null
              const d = payload[0].payload as DayRow
              return (
                <div className="card px-3 py-2 text-xs shadow-md">
                  <div className="text-muted mb-1">{d.label}</div>
                  <TipRow color="var(--viz-ok)" value={d.succeeded} label="succeeded" />
                  <TipRow color="var(--viz-failed)" value={d.failed} label="failed" />
                  {d.processing > 0 && <div className="text-muted mt-0.5">{d.processing} in progress</div>}
                </div>
              )
            }}
          />
          <Bar dataKey="succeeded" stackId="volume" fill="var(--viz-ok)" stroke="var(--surface)" strokeWidth={2} maxBarSize={24} isAnimationActive={false} shape={(p: unknown) => segment(p, 'succeeded')} />
          <Bar dataKey="failed" stackId="volume" fill="var(--viz-failed)" stroke="var(--surface)" strokeWidth={2} maxBarSize={24} isAnimationActive={false} shape={(p: unknown) => segment(p, 'failed')} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}

function TipRow({ color, value, label }: { color: string; value: number; label: string }) {
  return (
    <div className="flex items-center gap-2">
      <span className="h-0.5 w-3 rounded-full" style={{ background: color }} />
      <span className="font-semibold tabular-nums">{value}</span>
      <span className="text-muted">{label}</span>
    </div>
  )
}

function FailedList() {
  const qc = useQueryClient()
  const [page, setPage] = useState(1)
  const [notice, setNotice] = useState('')
  const [error, setError] = useState('')
  const failed = useQuery({ queryKey: extractionKeys.failed(page), queryFn: () => extractionApi.failed(page, FAILED_PAGE), refetchInterval: 15_000, placeholderData: (p) => p })
  const retry = useMutation({
    mutationFn: (id: string) => extractionApi.retry(id),
    onSuccess: () => {
      setError('')
      setNotice('Extraction queued. The receipt leaves this list once the user backend picks it up.')
      qc.invalidateQueries({ queryKey: extractionKeys.all })
      window.setTimeout(() => qc.invalidateQueries({ queryKey: extractionKeys.all }), 1500)
    },
    onError: (e) => { setNotice(''); setError(e instanceof ApiRequestError ? e.message : 'Retry failed') },
  })
  const data = failed.data
  const pages = Math.max(1, Math.ceil((data?.total ?? 0) / FAILED_PAGE))

  return (
    <section className="card mt-4">
      <div className="flex items-center justify-between gap-2 border-b border-[var(--border)] px-4 py-3">
        <h2 className="text-base font-semibold">Failed extractions{data ? ` (${data.total.toLocaleString('en-IN')})` : ''}</h2>
      </div>
      {(notice || error) && <div className="px-4 pt-3">{error ? <Alert>{error}</Alert> : <Alert kind="success">{notice}</Alert>}</div>}
      {failed.error && <div className="p-4"><Alert>Could not load failed receipts.</Alert></div>}
      {!data ? (
        <div className="grid gap-2 p-4">{Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-14" />)}</div>
      ) : data.receipts.length === 0 ? (
        <EmptyState icon={CircleCheck} title="No failed extractions" description="Receipts that fail after every retry show up here." />
      ) : (
        <ul className="divide-y divide-[var(--border)]">
          {data.receipts.map((r) => (
            <FailedRow key={r.id} r={r} pending={retry.isPending && retry.variables === r.id} onRetry={() => retry.mutate(r.id)} />
          ))}
        </ul>
      )}
      {pages > 1 && (
        <div className="flex items-center justify-between border-t border-[var(--border)] px-4 py-3 text-sm">
          <span className="text-muted">Page {page} of {pages}</span>
          <div className="flex gap-2">
            <Button variant="secondary" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>Previous</Button>
            <Button variant="secondary" size="sm" disabled={page >= pages} onClick={() => setPage((p) => p + 1)}>Next</Button>
          </div>
        </div>
      )}
    </section>
  )
}

function FailedRow({ r, pending, onRetry }: { r: FailedReceipt; pending: boolean; onRetry: () => void }) {
  return (
    <li className="flex flex-col gap-2 px-4 py-3 sm:flex-row sm:items-start">
      <div className="min-w-0 flex-1">
        <div className="text-sm font-medium">
          {r.merchant || 'Untitled receipt'} <span className="text-muted text-xs font-normal">· {plural(r.imageCount, 'file')}</span>
        </div>
        <div className="text-muted text-xs break-all">
          {r.userEmail || r.userId} · failed {when(r.failedAt)} · {plural(r.attempts, 'attempt')}{r.model ? ` · ${r.model}` : ''}
        </div>
        <div className="mt-1 flex items-start gap-1.5 text-xs text-red-700 dark:text-red-300">
          <AlertTriangle className="mt-px size-3.5 shrink-0" />
          <span className="font-mono break-words">{r.error || 'No error recorded'}</span>
        </div>
      </div>
      <Button variant="secondary" size="sm" className="self-start" loading={pending} onClick={onRetry} aria-label={`Retry receipt ${r.id}`}>
        <RotateCw className="size-4" /> Retry
      </Button>
    </li>
  )
}
