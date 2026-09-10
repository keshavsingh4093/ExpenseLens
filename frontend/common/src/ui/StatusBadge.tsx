import { clsx } from 'clsx'
import type { ReceiptStatus } from '../types'

const styles: Record<ReceiptStatus, [string, string]> = {
  processing: ['Extracting…', 'bg-blue-50 text-blue-700 dark:bg-blue-950/40 dark:text-blue-300'],
  review: ['Needs review', 'bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300'],
  confirmed: ['Confirmed', 'bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300'],
  failed: ['Failed', 'bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300'],
}

export function StatusBadge({ status, className }: { status: ReceiptStatus; className?: string }) {
  const [label, cls] = styles[status]
  return <span className={clsx('inline-flex rounded-full px-2 py-0.5 text-xs font-medium', cls, className)}>{label}</span>
}
