import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Link } from 'react-router'
import { z } from 'zod'
import { Button } from '@/components/ui/Button'
import { Field, Input } from '@/components/ui/Field'
import { Alert } from '@/components/ui/Alert'
import { authApi } from './api'
import { AuthLayout } from './AuthLayout'

const schema = z.object({ email: z.string().email('Enter a valid email') })
type Form = z.infer<typeof schema>

export function ForgotPasswordPage() {
  const [sent, setSent] = useState(false)
  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<Form>({ resolver: zodResolver(schema) })
  return (
    <AuthLayout title="Reset your password" subtitle="We'll email you a reset link">
      {sent ? (
        <Alert kind="success">If that email exists, a reset link is on its way. Check your inbox.</Alert>
      ) : (
        <form onSubmit={handleSubmit(async (v) => { await authApi.forgotPassword(v.email); setSent(true) })} className="grid gap-4" noValidate>
          <Field label="Email" error={errors.email?.message}>
            <Input type="email" autoComplete="email" invalid={!!errors.email} {...register('email')} />
          </Field>
          <Button type="submit" loading={isSubmitting}>Send reset link</Button>
        </form>
      )}
      <p className="text-muted mt-5 text-center text-sm">
        <Link to="/login" className="text-brand-700 hover:underline">Back to sign in</Link>
      </p>
    </AuthLayout>
  )
}
