import { Tag } from 'lucide-react'
import { clsx } from 'clsx'
import type { Category } from './types'
import { categoryColor } from './colors'
import { categoryIcons } from './icons'

export function CategoryIcon({ category, size = 'md', className }: { category?: Category; size?: 'sm' | 'md' | 'lg'; className?: string }) {
  const Icon = categoryIcons[category?.icon ?? 'tag'] ?? Tag
  const color = categoryColor(category)
  const dim = size === 'sm' ? 'size-7' : size === 'lg' ? 'size-12' : 'size-9'
  const ic = size === 'sm' ? 'size-3.5' : size === 'lg' ? 'size-6' : 'size-4.5'
  return (
    <span className={clsx('grid shrink-0 place-items-center rounded-lg', dim, className)} style={{ background: `${color}22`, color }}>
      <Icon className={ic} />
    </span>
  )
}

export function CategoryBadge({ category }: { category?: Category }) {
  const color = categoryColor(category)
  return (
    <span className="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium" style={{ background: `${color}22`, color }}>
      {category?.name ?? 'Uncategorized'}
    </span>
  )
}
