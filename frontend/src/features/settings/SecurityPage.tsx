import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { Button } from '@/components/ui/Button'
import { Field, Input } from '@/components/ui/Field'
import { Alert } from '@/components/ui/Alert'
import { useAuth } from '@/hooks/useAuth'
import { authApi } from '@/features/auth/api'
import { ApiRequestError } from '@/lib/api/client'
import { SettingsSection } from './SettingsLayout'

const schema = z
  .object({ currentPassword: z.string(), newPassword: z.string().min(8, 'At least 8 characters'), confirm: z.string() })
  .refine((v) => v.newPassword === v.confirm, { path: ['confirm'], message: 'Passwords do not match' })
type Form = z.infer<typeof schema>

const labels: Record<string, string> = { google: 'Google', microsoft: 'Microsoft', apple: 'Apple' }

export function SecurityPage() {
  const { user, setUser } = useAuth()
  const [saved, setSaved] = useState(false)
  const { register, handleSubmit, reset, setError, formState: { errors, isSubmitting } } = useForm<Form>({ resolver: zodResolver(schema) })
  if (!user) return null

  const onSubmit = async (v: Form) => {
    try {
      await authApi.changePassword({ currentPassword: v.currentPassword, newPassword: v.newPassword })
      setUser(await authApi.me())
      reset()
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch (e) {
      if (e instanceof ApiRequestError && e.fields) {
        for (const [k, msg] of Object.entries(e.fields)) setError(k as keyof Form, { message: msg })
      } else setError('root', { message: e instanceof ApiRequestError ? e.message : 'Could not save' })
    }
  }

  return (
    <div className="grid gap-8">
      <SettingsSection title={user.hasPassword ? 'Change password' : 'Set a password'}>
        <form onSubmit={handleSubmit(onSubmit)} className="grid gap-4" noValidate>
          {errors.root && <Alert>{errors.root.message}</Alert>}
          {saved && <Alert kind="success">Password updated.</Alert>}
          {user.hasPassword && (
            <Field label="Current password" error={errors.currentPassword?.message}>
              <Input type="password" autoComplete="current-password" invalid={!!errors.currentPassword} {...register('currentPassword')} />
            </Field>
          )}
          <Field label="New password" error={errors.newPassword?.message}>
            <Input type="password" autoComplete="new-password" invalid={!!errors.newPassword} {...register('newPassword')} />
          </Field>
          <Field label="Confirm new password" error={errors.confirm?.message}>
            <Input type="password" autoComplete="new-password" invalid={!!errors.confirm} {...register('confirm')} />
          </Field>
          <div><Button type="submit" loading={isSubmitting}>Update password</Button></div>
        </form>
      </SettingsSection>
      <SettingsSection title="Connected accounts">
        {user.providers.length === 0 ? (
          <p className="text-muted text-sm">No social accounts linked. Sign in with Google, Microsoft, or Apple using the same email to link one.</p>
        ) : (
          <ul className="grid gap-2">
            {user.providers.map((p) => (
              <li key={p} className="flex items-center justify-between rounded-lg border border-[var(--border)] px-3 py-2 text-sm">
                <span>{labels[p] ?? p}</span>
                <span className="text-xs text-brand-700">Connected</span>
              </li>
            ))}
          </ul>
        )}
      </SettingsSection>
    </div>
  )
}
