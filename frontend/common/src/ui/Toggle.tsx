import { clsx } from 'clsx'

export function Toggle({ checked, onChange, label, description }: { checked: boolean; onChange: (v: boolean) => void; label: string; description?: string }) {
  return (
    <div className="flex items-center justify-between gap-4 py-3">
      <div>
        <div className="text-sm font-medium">{label}</div>
        {description && <div className="text-muted text-xs">{description}</div>}
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        onClick={() => onChange(!checked)}
        className={clsx(
          'relative h-6 w-11 shrink-0 rounded-full transition-colors',
          checked ? 'bg-brand-600' : 'bg-gray-300 dark:bg-gray-600',
        )}
      >
        <span className={clsx('absolute top-0.5 size-5 rounded-full bg-white transition-all', checked ? 'left-5.5' : 'left-0.5')} />
      </button>
    </div>
  )
}
