import { clsx } from 'clsx'
import type { ButtonHTMLAttributes } from 'react'

export function Chip({ active, className, ...rest }: ButtonHTMLAttributes<HTMLButtonElement> & { active?: boolean }) {
  return (
    <button
      type="button"
      className={clsx(
        'h-8 shrink-0 rounded-full border px-3 text-xs font-medium whitespace-nowrap transition-colors',
        active
          ? 'border-brand-600 bg-brand-600 text-white'
          : 'border-[var(--border)] bg-[var(--surface)] hover:border-brand-400 hover:text-brand-700',
        className,
      )}
      {...rest}
    />
  )
}
