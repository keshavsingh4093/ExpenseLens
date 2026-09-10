import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { Link, useLocation, useNavigate, useSearchParams } from 'react-router'
import { z } from 'zod'
import { Button } from '@/components/ui/Button'
import { Field, Input } from '@/components/ui/Field'
import { Alert } from '@/components/ui/Alert'
import { useAuth } from '@/hooks/useAuth'
import { ApiRequestError } from '@/lib/api/client'
import { authApi } from './api'
import { AuthLayout } from './AuthLayout'
import { OAuthButtons } from './OAuthButtons'

const schema = z.object({ email: z.string().email('Enter a valid email'), password: z.string().min(1, 'Enter your password') })
type Form = z.infer<typeof schema>

const oauthErrors: Record<string, string> = {
  email_unverified: 'That email already has an account. Sign in with your password, then link the provider in Settings.',
  access_denied: 'Sign-in was cancelled.',
}

export function LoginPage() {
  const { setSession } = useAuth()
  const nav = useNavigate()
  const loc = useLocation()
  const [params] = useSearchParams()
  const oauthError = params.get('error')
  const { register, handleSubmit, setError, formState: { errors, isSubmitting } } = useForm<Form>({ resolver: zodResolver(schema) })

  const onSubmit = async (v: Form) => {
    try {
      setSession(await authApi.login(v))
      nav((loc.state as { from?: string })?.from ?? '/', { replace: true })
    } catch (e) {
      setError('root', { message: e instanceof ApiRequestError ? e.message : 'Something went wrong' })
    }
  }

  return (
    <AuthLayout title="Welcome back" subtitle="Sign in to track your expenses">
      {oauthError && <div className="mb-4"><Alert>{oauthErrors[oauthError] ?? `Sign-in failed (${oauthError}).`}</Alert></div>}
      <form onSubmit={handleSubmit(onSubmit)} className="grid gap-4" noValidate>
        {errors.root && <Alert>{errors.root.message}</Alert>}
        <Field label="Email" error={errors.email?.message}>
          <Input type="email" autoComplete="email" invalid={!!errors.email} {...register('email')} />
        </Field>
        <Field label="Password" error={errors.password?.message}>
          <Input type="password" autoComplete="current-password" invalid={!!errors.password} {...register('password')} />
        </Field>
        <div className="-mt-2 text-right">
          <Link to="/forgot-password" className="text-xs text-brand-700 hover:underline">Forgot password?</Link>
        </div>
        <Button type="submit" loading={isSubmitting}>Sign in</Button>
      </form>
      <OAuthButtons />
      <p className="text-muted mt-5 text-center text-sm">
        New here? <Link to="/register" className="text-brand-700 hover:underline">Create an account</Link>
      </p>
    </AuthLayout>
  )
}
