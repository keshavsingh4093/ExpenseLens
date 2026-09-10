import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { Button } from '@/components/ui/Button'
import { Field, Input } from '@/components/ui/Field'
import { Alert } from '@/components/ui/Alert'
import { useAuth } from '@/hooks/useAuth'
import { meApi } from '@/features/auth/api'
import { ApiRequestError } from '@/lib/api/client'
import { SettingsSection } from './SettingsLayout'

const schema = z.object({ name: z.string().trim().min(1, 'Enter your name').max(80) })
type Form = z.infer<typeof schema>

export function ProfilePage() {
  const { user, setUser } = useAuth()
  const [saved, setSaved] = useState(false)
  const { register, handleSubmit, setError, formState: { errors, isSubmitting, isDirty } } = useForm<Form>({
    resolver: zodResolver(schema),
    defaultValues: { name: user?.name ?? '' },
  })
  if (!user) return null
  const initials = user.name.split(' ').map((s) => s[0]).join('').slice(0, 2).toUpperCase()

  const onSubmit = async (v: Form) => {
    try {
      setUser(await meApi.update(v))
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch (e) {
      setError('root', { message: e instanceof ApiRequestError ? e.message : 'Could not save' })
    }
  }

  return (
    <SettingsSection title="Profile">
      <div className="mb-6 flex items-center gap-4">
        {user.avatarUrl ? (
          <img src={user.avatarUrl} alt="" className="size-16 rounded-full object-cover" />
        ) : (
          <div className="grid size-16 place-items-center rounded-full bg-brand-100 text-lg font-semibold text-brand-700">{initials}</div>
        )}
        <div>
          <div className="font-medium">{user.name}</div>
          <div className="text-muted text-sm">{user.email}</div>
        </div>
      </div>
      <form onSubmit={handleSubmit(onSubmit)} className="grid gap-4" noValidate>
        {errors.root && <Alert>{errors.root.message}</Alert>}
        {saved && <Alert kind="success">Profile saved.</Alert>}
        <Field label="Full name" error={errors.name?.message}>
          <Input invalid={!!errors.name} {...register('name')} />
        </Field>
        <Field label="Email" hint={user.emailVerified ? 'Verified' : 'Not verified'}>
          <Input value={user.email} disabled readOnly />
        </Field>
        <div>
          <Button type="submit" loading={isSubmitting} disabled={!isDirty}>Save changes</Button>
        </div>
      </form>
    </SettingsSection>
  )
}
