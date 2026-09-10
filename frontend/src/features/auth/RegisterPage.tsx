import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { Link, useNavigate } from 'react-router'
import { z } from 'zod'
import { Button } from '@/components/ui/Button'
import { Field, Input } from '@/components/ui/Field'
import { Alert } from '@/components/ui/Alert'
import { useAuth } from '@/hooks/useAuth'
import { ApiRequestError } from '@/lib/api/client'
import { authApi } from './api'
import { AuthLayout } from './AuthLayout'
import { OAuthButtons } from './OAuthButtons'

const schema = z.object({
  name: z.string().trim().min(1, 'Enter your name').max(80),
  email: z.string().email('Enter a valid email'),
  password: z.string().min(8, 'At least 8 characters'),
})
type Form = z.infer<typeof schema>

export function RegisterPage() {
  const { setSession } = useAuth()
  const nav = useNavigate()
  const { register, handleSubmit, setError, formState: { errors, isSubmitting } } = useForm<Form>({ resolver: zodResolver(schema) })

  const onSubmit = async (v: Form) => {
    try {
      setSession(await authApi.register(v))
      nav('/', { replace: true })
    } catch (e) {
      if (e instanceof ApiRequestError && e.fields) {
        for (const [k, msg] of Object.entries(e.fields)) setError(k as keyof Form, { message: msg })
      } else {
        setError('root', { message: e instanceof ApiRequestError ? e.message : 'Something went wrong' })
      }
    }
  }

  return (
    <AuthLayout title="Create your account" subtitle="Start tracking receipts in minutes">
      <form onSubmit={handleSubmit(onSubmit)} className="grid gap-4" noValidate>
        {errors.root && <Alert>{errors.root.message}</Alert>}
        <Field label="Full name" error={errors.name?.message}>
          <Input autoComplete="name" invalid={!!errors.name} {...register('name')} />
        </Field>
        <Field label="Email" error={errors.email?.message}>
          <Input type="email" autoComplete="email" invalid={!!errors.email} {...register('email')} />
        </Field>
        <Field label="Password" error={errors.password?.message} hint="At least 8 characters">
          <Input type="password" autoComplete="new-password" invalid={!!errors.password} {...register('password')} />
        </Field>
        <Button type="submit" loading={isSubmitting}>Create account</Button>
      </form>
      <OAuthButtons />
      <p className="text-muted mt-5 text-center text-sm">
        Already have an account? <Link to="/login" className="text-brand-700 hover:underline">Sign in</Link>
      </p>
    </AuthLayout>
  )
}
