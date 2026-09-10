import { useState } from 'react'
import { Button } from '@/components/ui/Button'
import { Field, Select } from '@/components/ui/Field'
import { Alert } from '@/components/ui/Alert'
import { useAuth } from '@/hooks/useAuth'
import { useTheme } from '@/hooks/useTheme'
import { meApi } from '@/features/auth/api'
import type { Preferences } from '@/features/auth/types'
import { SettingsSection } from './SettingsLayout'

const currencies = [['INR', 'INR (₹)'], ['USD', 'USD ($)'], ['EUR', 'EUR (€)'], ['GBP', 'GBP (£)'], ['AED', 'AED (د.إ)'], ['SGD', 'SGD (S$)']]
const languages = [['en', 'English'], ['hi', 'हिन्दी']]
const timezones = ['Asia/Kolkata', 'Asia/Dubai', 'Asia/Singapore', 'Europe/London', 'Europe/Berlin', 'America/New_York', 'America/Los_Angeles', 'UTC']

export function PreferencesPage() {
  const { user, setUser } = useAuth()
  const { setTheme } = useTheme()
  const [form, setForm] = useState<Omit<Preferences, 'notifications'>>(() => {
    const p = user!.preferences
    return { currency: p.currency, language: p.language, theme: p.theme, timezone: p.timezone }
  })
  const [state, setState] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle')
  if (!user) return null
  const set = <K extends keyof typeof form>(k: K, v: (typeof form)[K]) => setForm((f) => ({ ...f, [k]: v }))

  const save = async () => {
    setState('saving')
    try {
      setUser(await meApi.updatePreferences(form))
      setTheme(form.theme)
      setState('saved')
      setTimeout(() => setState('idle'), 2000)
    } catch {
      setState('error')
    }
  }

  return (
    <SettingsSection title="Preferences">
      <div className="grid gap-4">
        {state === 'saved' && <Alert kind="success">Preferences saved.</Alert>}
        {state === 'error' && <Alert>Could not save preferences.</Alert>}
        <Field label="Currency">
          <Select value={form.currency} onChange={(e) => set('currency', e.target.value)}>
            {currencies.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
          </Select>
        </Field>
        <Field label="Language">
          <Select value={form.language} onChange={(e) => set('language', e.target.value)}>
            {languages.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
          </Select>
        </Field>
        <Field label="Theme">
          <Select value={form.theme} onChange={(e) => set('theme', e.target.value as Preferences['theme'])}>
            <option value="system">System</option>
            <option value="light">Light</option>
            <option value="dark">Dark</option>
          </Select>
        </Field>
        <Field label="Timezone">
          <Select value={form.timezone} onChange={(e) => set('timezone', e.target.value)}>
            {timezones.map((tz) => <option key={tz} value={tz}>{tz}</option>)}
          </Select>
        </Field>
        <div><Button onClick={save} loading={state === 'saving'}>Save preferences</Button></div>
      </div>
    </SettingsSection>
  )
}
