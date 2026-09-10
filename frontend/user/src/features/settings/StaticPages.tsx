import { useState } from 'react'
import { Link } from 'react-router'
import { Download, FileText } from 'lucide-react'
import { Button } from '@common/ui/Button'
import { Alert } from '@common/ui/Alert'
import { reportsApi } from '@/features/reports/api'
import { SettingsSection } from './SettingsLayout'

export function ExportPage() {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const exportAll = async () => {
    setBusy(true); setError('')
    try { await reportsApi.exportAll() } catch (e) { setError((e as Error).message) } finally { setBusy(false) }
  }
  return (
    <SettingsSection title="Export data">
      {error && <div className="mb-3"><Alert>{error}</Alert></div>}
      <p className="text-muted text-sm">Download your receipts as CSV or a PDF summary for any period from the Reports screen.</p>
      <div className="mt-3"><Link to="/reports"><Button variant="secondary"><FileText className="size-4" /> Open Reports</Button></Link></div>
      <h3 className="mt-6 mb-1 text-sm font-semibold">Everything as JSON</h3>
      <p className="text-muted text-sm">One file with your profile, categories, receipts (with image links), budgets, warranties and notifications.</p>
      <div className="mt-3"><Button onClick={exportAll} loading={busy}><Download className="size-4" /> Export all data</Button></div>
    </SettingsSection>
  )
}

export function HelpPage() {
  return (
    <SettingsSection title="Help & support">
      <p className="text-muted text-sm">Questions or issues? Email support@expenselens.local.</p>
    </SettingsSection>
  )
}

export function AboutPage() {
  return (
    <SettingsSection title="About">
      <p className="text-sm">ExpenseLens v0.1.0</p>
      <p className="text-muted mt-1 text-sm">Receipt-first expense tracking. © 2026 ExpenseLens.</p>
    </SettingsSection>
  )
}
