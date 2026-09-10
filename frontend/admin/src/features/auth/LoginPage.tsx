import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { useLocation, useNavigate } from 'react-router'
import { z } from 'zod'
import { ShieldCheck } from 'lucide-react'
import { Button } from '@common/ui/Button'
import { Field, Input } from '@common/ui/Field'
import { Alert } from '@common/ui/Alert'
import { ApiRequestError } from '@common/api/client'
import { useAuth } from './useAuth'
import { adminAuthApi } from './api'

const schema = z.object({ email: z.string().email('Enter a valid email'), password: z.string().min(1, 'Enter your password') })
type Form = z.infer<typeof schema>

export function LoginPage() {
  const { setSession } = useAuth()
  const nav = useNavigate()
  const loc = useLocation()
  const { register, handleSubmit, setError, formState: { errors, isSubmitting } } = useForm<Form>({ resolver: zodResolver(schema) })

  const onSubmit = async (v: Form) => {
    try {
      setSession(await adminAuthApi.login(v))
      nav((loc.state as { from?: string })?.from ?? '/', { replace: true })
    } catch (e) {
      setError('root', { message: e instanceof ApiRequestError ? e.message : 'Something went wrong' })
    }
  }

  return (
    <div className="grid min-h-dvh place-items-center px-4 py-8">
      <div className="w-full max-w-sm">
        <div className="mb-6 flex items-center justify-center gap-2">
          <span className="grid size-9 place-items-center rounded-lg bg-brand-600 text-white">
            <ShieldCheck className="size-5" />
          </span>
          <span className="text-xl font-semibold">ExpenseLens Admin</span>
        </div>
        <div className="card p-6">
          <h1 className="text-lg font-semibold">Sign in</h1>
          <p className="text-muted mt-1 text-sm">Admin accounts only.</p>
          <form onSubmit={handleSubmit(onSubmit)} className="mt-5 grid gap-4" noValidate>
            {errors.root && <Alert>{errors.root.message}</Alert>}
            <Field label="Email" error={errors.email?.message}>
              <Input type="email" autoComplete="username" invalid={!!errors.email} {...register('email')} />
            </Field>
            <Field label="Password" error={errors.password?.message}>
              <Input type="password" autoComplete="current-password" invalid={!!errors.password} {...register('password')} />
            </Field>
            <Button type="submit" loading={isSubmitting}>Sign in</Button>
          </form>
        </div>
      </div>
    </div>
  )
}
