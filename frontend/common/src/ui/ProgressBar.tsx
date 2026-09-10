import { clsx } from 'clsx'

/** Budget-style progress: green under 80%, amber to 100%, red over. */
export function ProgressBar({ percent, className, label }: { percent: number; className?: string; label?: string }) {
  const p = Math.max(0, Math.min(100, percent))
  const tone = percent >= 100 ? 'bg-red-500' : percent >= 80 ? 'bg-amber-500' : 'bg-brand-600'
  return (
    <div
      role="progressbar"
      aria-valuenow={Math.round(percent)}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-label={label}
      className={clsx('h-2 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700', className)}
    >
      <div className={clsx('h-full rounded-full transition-[width]', tone)} style={{ width: `${p}%` }} />
    </div>
  )
}
