import { useState } from 'react'
import { FileSpreadsheet, FileText, Download } from 'lucide-react'
import { Button } from '@common/ui/Button'
import { Chip } from '@common/ui/Chip'
import { Field, Input, Select } from '@common/ui/Field'
import { Alert } from '@common/ui/Alert'
import type { RangeName } from '@common/analytics/types'
import { currentMonth, monthLabel, shiftMonth } from '@/features/budgets/api'
import { reportsApi, type ReportFormat } from './api'

const ranges: [RangeName | 'custom', string][] = [['month', 'This month'], ['30d', 'Last 30 days'], ['3m', 'Last 3 months'], ['year', 'This year'], ['all', 'All time'], ['custom', 'Custom']]

export function ReportsPage() {
  const [range, setRange] = useState<RangeName | 'custom'>('month')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [month, setMonth] = useState(currentMonth())
  const [busy, setBusy] = useState<string | null>(null)
  const [error, setError] = useState('')

  const run = async (key: string, fn: () => Promise<void>) => {
    setBusy(key); setError('')
    try { await fn() } catch (e) { setError((e as Error).message || 'Download failed') } finally { setBusy(null) }
  }
  const params = range === 'custom' ? { from, to } : { range }
  const download = (format: ReportFormat) => run(format, () => reportsApi.export(format, params))
  const monthly = () => {
    const [y, m] = month.split('-').map(Number)
    const last = new Date(y, m, 0).getDate()
    return run('monthly', () => reportsApi.export('pdf', { from: `${month}-01`, to: `${month}-${String(last).padStart(2, '0')}` }))
  }
  const months = Array.from({ length: 12 }, (_, i) => shiftMonth(currentMonth(), -i))

  return (
    <div className="mx-auto max-w-3xl">
      <div className="mb-4">
        <h1 className="text-2xl font-semibold">Reports</h1>
        <p className="text-muted text-sm">Export your receipts as a spreadsheet or a PDF summary.</p>
      </div>
      {error && <div className="mb-3"><Alert>{error}</Alert></div>}

      <section className="card mb-4 p-4">
        <h2 className="mb-3 text-sm font-semibold">Export receipts</h2>
        <div className="mb-3 flex gap-2 overflow-x-auto pb-1" role="tablist" aria-label="Range">
          {ranges.map(([v, l]) => <Chip key={v} role="tab" aria-selected={range === v} active={range === v} onClick={() => setRange(v)}>{l}</Chip>)}
        </div>
        {range === 'custom' && (
          <div className="mb-3 grid grid-cols-2 gap-3">
            <Field label="From"><Input type="date" name="from" value={from} onChange={(e) => setFrom(e.target.value)} /></Field>
            <Field label="To"><Input type="date" name="to" value={to} onChange={(e) => setTo(e.target.value)} /></Field>
          </div>
        )}
        <div className="flex flex-wrap gap-2">
          <Button onClick={() => download('csv')} loading={busy === 'csv'} disabled={!!busy || (range === 'custom' && !from && !to)}><FileSpreadsheet className="size-4" /> Download CSV</Button>
          <Button variant="secondary" onClick={() => download('pdf')} loading={busy === 'pdf'} disabled={!!busy || (range === 'custom' && !from && !to)}><FileText className="size-4" /> Download PDF summary</Button>
        </div>
        <p className="text-muted mt-3 text-xs">CSV includes date, merchant, category, amounts, payment method, GSTIN, items, tags and notes. The PDF has totals, category and merchant breakdowns and a daily chart.</p>
      </section>

      <section className="card p-4">
        <h2 className="mb-3 text-sm font-semibold">Monthly summary</h2>
        <div className="flex flex-wrap items-end gap-3">
          <Field label="Month">
            <Select name="month" value={month} onChange={(e) => setMonth(e.target.value)} className="w-auto!">
              {months.map((m) => <option key={m} value={m}>{monthLabel(m)}</option>)}
            </Select>
          </Field>
          <Button onClick={monthly} loading={busy === 'monthly'} disabled={!!busy}><Download className="size-4" /> Monthly PDF</Button>
        </div>
      </section>
    </div>
  )
}
