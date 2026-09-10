import { Toggle } from '@/components/ui/Toggle'
import { useAuth } from '@/hooks/useAuth'
import { meApi } from '@/features/auth/api'
import { SettingsSection } from './SettingsLayout'

export function NotificationsSettingsPage() {
  const { user, setUser } = useAuth()
  if (!user) return null
  const n = user.preferences.notifications
  const update = async (patch: Partial<typeof n>) => setUser(await meApi.updatePreferences({ notifications: patch }))
  return (
    <SettingsSection title="Notifications">
      <div className="divide-y divide-[var(--border)]">
        <Toggle label="Budget alerts" description="Warn me at 80% and 100% of a category budget" checked={n.budgetAlerts} onChange={(v) => update({ budgetAlerts: v })} />
        <Toggle label="Warranty reminders" description="Remind me 30 and 7 days before a warranty expires" checked={n.warrantyReminders} onChange={(v) => update({ warrantyReminders: v })} />
        <Toggle label="Weekly summary" description="A short spending recap every Monday" checked={n.weeklySummary} onChange={(v) => update({ weeklySummary: v })} />
      </div>
    </SettingsSection>
  )
}
