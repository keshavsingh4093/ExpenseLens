import { useState } from 'react'
import { useNavigate } from 'react-router'
import { Download } from 'lucide-react'
import { Button } from '@common/ui/Button'
import { Alert } from '@common/ui/Alert'
import { useAuth } from '@/features/auth/useAuth'
import { meApi } from '@/features/auth/api'
import { reportsApi } from '@/features/reports/api'
import { SettingsSection } from './SettingsLayout'

export function DataPrivacyPage() {
  const { logout } = useAuth()
  const nav = useNavigate()
  const [confirm, setConfirm] = useState('')
  const [busy, setBusy] = useState(false)
  const [exporting, setExporting] = useState(false)
  const exportAll = async () => {
    setExporting(true)
    try { await reportsApi.exportAll() } finally { setExporting(false) }
  }
  const del = async () => {
    setBusy(true)
    try {
      await meApi.deleteAccount()
      await logout()
      nav('/login', { replace: true })
    } finally {
      setBusy(false)
    }
  }
  return (
    <div className="grid gap-8">
      <SettingsSection title="Your data">
        <p className="text-muted text-sm">Receipts, images, budgets and warranties are stored only for your account. You can download everything as a single JSON file at any time.</p>
        <div className="mt-3"><Button variant="secondary" onClick={exportAll} loading={exporting}><Download className="size-4" /> Export all data</Button></div>
      </SettingsSection>
      <SettingsSection title="Delete account">
        <Alert>This permanently deletes your account and all receipts. This cannot be undone.</Alert>
        <label className="mt-4 block text-sm">
          Type <strong>DELETE</strong> to confirm
          <input className="mt-1 h-10 w-full rounded-lg border border-[var(--border)] bg-[var(--surface)] px-3 text-sm" value={confirm} onChange={(e) => setConfirm(e.target.value)} />
        </label>
        <div className="mt-3"><Button variant="danger" disabled={confirm !== 'DELETE'} loading={busy} onClick={del}>Delete my account</Button></div>
      </SettingsSection>
    </div>
  )
}
