import { ArrowLeft } from 'lucide-react'
import { useNavigate } from 'react-router'
import type { ReactNode } from 'react'

export function PageHeader({ title, subtitle, back, actions }: { title: string; subtitle?: string; back?: string | true; actions?: ReactNode }) {
  const nav = useNavigate()
  return (
    <div className="mb-4 flex flex-wrap items-center gap-3">
      {back && (
        <button
          type="button"
          aria-label="Back"
          onClick={() => (back === true ? nav(-1) : nav(back))}
          className="text-muted grid size-9 place-items-center rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800"
        >
          <ArrowLeft className="size-5" />
        </button>
      )}
      <div className="min-w-0 flex-1">
        <h1 className="truncate text-2xl font-semibold">{title}</h1>
        {subtitle && <p className="text-muted text-sm">{subtitle}</p>}
      </div>
      {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
    </div>
  )
}
