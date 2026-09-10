import { useQuery } from '@tanstack/react-query'
import { authApi } from './api'

const labels: Record<string, string> = { google: 'Google', microsoft: 'Microsoft', apple: 'Apple' }

function Icon({ p }: { p: string }) {
  if (p === 'google')
    return (
      <svg viewBox="0 0 24 24" className="size-4" aria-hidden>
        <path fill="#EA4335" d="M12 10.2v3.9h5.5c-.2 1.3-1.6 3.9-5.5 3.9-3.3 0-6-2.7-6-6s2.7-6 6-6c1.9 0 3.1.8 3.9 1.5l2.6-2.6C16.9 3.4 14.7 2.4 12 2.4 6.7 2.4 2.4 6.7 2.4 12s4.3 9.6 9.6 9.6c5.5 0 9.2-3.9 9.2-9.4 0-.6-.1-1.1-.2-1.6H12z" />
      </svg>
    )
  if (p === 'microsoft')
    return (
      <svg viewBox="0 0 24 24" className="size-4" aria-hidden>
        <path fill="#F25022" d="M2 2h9.5v9.5H2z" /><path fill="#7FBA00" d="M12.5 2H22v9.5h-9.5z" />
        <path fill="#00A4EF" d="M2 12.5h9.5V22H2z" /><path fill="#FFB900" d="M12.5 12.5H22V22h-9.5z" />
      </svg>
    )
  return (
    <svg viewBox="0 0 24 24" className="size-4 fill-current" aria-hidden>
      <path d="M16.4 12.6c0-2.5 2-3.6 2.1-3.7-1.2-1.7-3-1.9-3.6-2-1.5-.2-3 .9-3.8.9-.8 0-2-.9-3.3-.8-1.7 0-3.2 1-4.1 2.5-1.8 3.1-.5 7.6 1.3 10.1.9 1.2 1.9 2.6 3.2 2.5 1.3-.1 1.8-.8 3.3-.8s2 .8 3.3.8c1.4 0 2.3-1.2 3.1-2.5.6-.9 1-1.8 1.4-2.8-.1 0-2.9-1.1-2.9-4.2zM14 5.3c.7-.8 1.2-2 1-3.2-1 0-2.2.7-2.9 1.5-.6.7-1.2 1.9-1.1 3 1.2.1 2.3-.6 3-1.3z" />
    </svg>
  )
}

export function OAuthButtons() {
  const { data } = useQuery({ queryKey: ['auth', 'providers'], queryFn: authApi.providers, staleTime: Infinity })
  const providers = data?.providers ?? []
  if (providers.length === 0) return null
  return (
    <div className="mt-5">
      <div className="text-muted relative my-4 text-center text-xs before:absolute before:top-1/2 before:left-0 before:w-full before:border-t before:border-[var(--border)]">
        <span className="relative bg-[var(--surface)] px-2">or continue with</span>
      </div>
      <div className="grid gap-2">
        {providers.map((p) => (
          <a
            key={p}
            href={authApi.oauthStartUrl(p)}
            className="flex h-10 items-center justify-center gap-2 rounded-lg border border-[var(--border)] text-sm font-medium hover:bg-brand-50 dark:hover:bg-brand-900/30"
          >
            <Icon p={p} /> {labels[p] ?? p}
          </a>
        ))}
      </div>
    </div>
  )
}
