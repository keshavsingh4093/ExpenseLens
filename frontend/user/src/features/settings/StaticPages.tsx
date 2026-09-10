import { SettingsSection } from './SettingsLayout'

export function ExportPage() {
  return (
    <SettingsSection title="Export data">
      <p className="text-muted text-sm">CSV and PDF export will be available once Reports ship (Phase 3).</p>
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
