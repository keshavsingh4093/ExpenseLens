import { clsx } from 'clsx'
import type { LucideIcon } from 'lucide-react'
import { TrendingDown, TrendingUp } from 'lucide-react'

/**
 * Stat tile: label · value · signed delta vs a named period.
 * For spending, up is bad: positive deltas render in the critical tone.
 */
export function StatTile({ label, value, delta, periodLabel, icon: Icon, upIsGood = false }: {
  label: string
  value: string
  delta?: number | null
  periodLabel?: string
  icon: LucideIcon
  upIsGood?: boolean
}) {
  const has = delta !== null && delta !== undefined
  const up = has && delta! >= 0
  const good = has && (up ? upIsGood : !upIsGood)
  return (
    <div className="card p-4">
      <div className="flex items-center gap-2">
        <span className="grid size-8 place-items-center rounded-lg bg-brand-50 text-brand-600 dark:bg-brand-900/30">
          <Icon className="size-4" />
        </span>
        <span className="text-muted text-xs">{label}</span>
      </div>
      <div className="mt-2 text-2xl font-semibold">{value}</div>
      {has ? (
        <div className={clsx('mt-1 flex items-center gap-1 text-xs', good ? 'text-brand-700 dark:text-brand-400' : 'text-red-600 dark:text-red-400')}>
          {up ? <TrendingUp className="size-3.5" /> : <TrendingDown className="size-3.5" />}
          <span className="font-medium">{up ? '+' : ''}{delta!.toFixed(1)}%</span>
          {periodLabel && <span className="text-muted font-normal">{periodLabel}</span>}
        </div>
      ) : (
        <div className="text-muted mt-1 text-xs">No comparison yet</div>
      )}
    </div>
  )
}
