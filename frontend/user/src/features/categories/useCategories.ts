import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'
import { categoriesApi } from './api'
import type { Category } from '@common/categories/types'

export const categoriesKey = ['categories'] as const

export function useCategories() {
  const q = useQuery({ queryKey: categoriesKey, queryFn: categoriesApi.list, staleTime: 5 * 60_000 })
  const byId = useMemo(() => {
    const m = new Map<string, Category>()
    for (const c of q.data?.categories ?? []) m.set(c.id, c)
    return m
  }, [q.data])
  return { categories: q.data?.categories ?? [], byId, isLoading: q.isLoading }
}
