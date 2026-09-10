import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { Link, useNavigate, useSearchParams } from 'react-router'
import { z } from 'zod'
import { Button } from '@/components/ui/Button'
import { Field, Input } from '@/components/ui/Field'
import { Alert } from '@/components/ui/Alert'
import { ApiRequestError } from '@/lib/api/client'
import { authApi } from './api'
import { AuthLayout } from './AuthLayout'

const schema = z
  .object({ password: z.string().min(8, 'At least 8 characters'), confirm: z.string() })
  .refine((v) => v.password === v.confirm, { path: ['confirm'], message: 'Passwords do not match' })
type Form = z.infer<typeof schema>

export function ResetPasswordPage() {
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''
  const nav = useNavigate()
  const { register, handleSubmit, setError, formState: { errors, isSubmitting } } = useForm<Form>({ resolver: zodResolver(schema) })

  const onSubmit = async (v: Form) => {
    try {
      await authApi.resetPassword(token, v.password)
      nav('/login', { replace: true })
    } catch (e) {
      setError('root', { message: e instanceof ApiRequestError ? e.message : 'Something went wrong' })
    }
  }

  if (!token) return <AuthLayout title="Invalid link"><Alert>This reset link is missing its token.</Alert></AuthLayout>
  return (
    <AuthLayout title="Choose a new password">
      <form onSubmit={handleSubmit(onSubmit)} className="grid gap-4" noValidate>
        {errors.root && <Alert>{errors.root.message}</Alert>}
        <Field label="New password" error={errors.password?.message}>
          <Input type="password" autoComplete="new-password" invalid={!!errors.password} {...register('password')} />
        </Field>
        <Field label="Confirm password" error={errors.confirm?.message}>
          <Input type="password" autoComplete="new-password" invalid={!!errors.confirm} {...register('confirm')} />
        </Field>
        <Button type="submit" loading={isSubmitting}>Set password</Button>
      </form>
      <p className="text-muted mt-5 text-center text-sm">
        <Link to="/login" className="text-brand-700 hover:underline">Back to sign in</Link>
      </p>
    </AuthLayout>
  )
}
