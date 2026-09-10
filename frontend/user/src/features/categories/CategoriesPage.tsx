import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Lock, Pencil, Plus, Tags, Trash2 } from 'lucide-react'
import { clsx } from 'clsx'
import { Button } from '@common/ui/Button'
import { EmptyState } from '@common/ui/EmptyState'
import { Modal, ConfirmDialog } from '@common/ui/Modal'
import { Field, Input } from '@common/ui/Field'
import { Alert } from '@common/ui/Alert'
import { CategoryIcon } from '@common/categories/CategoryIcon'
import { categoryIcons, categoryIconNames } from '@common/categories/icons'
import type { Category } from '@common/categories/types'
import { ApiRequestError } from '@common/api/client'
import { categoriesApi } from './api'
import { categoriesKey, useCategories } from './useCategories'
import { receiptKeys } from '@/features/receipts/api'

export function CategoriesPage() {
  const qc = useQueryClient()
  const { categories, isLoading } = useCategories()
  const [editing, setEditing] = useState<Category | 'new' | null>(null)
  const [deleting, setDeleting] = useState<Category | null>(null)
  const invalidate = () => { qc.invalidateQueries({ queryKey: categoriesKey }); qc.invalidateQueries({ queryKey: receiptKeys.all }) }
  const del = useMutation({ mutationFn: (id: string) => categoriesApi.remove(id), onSuccess: () => { setDeleting(null); invalidate() } })
  const system = categories.filter((c) => c.isSystem)
  const custom = categories.filter((c) => !c.isSystem)

  return (
    <div className="mx-auto max-w-3xl">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Categories</h1>
          <p className="text-muted text-sm">Default categories plus your own.</p>
        </div>
        <Button size="sm" onClick={() => setEditing('new')}><Plus className="size-4" /> Add category</Button>
      </div>

      <section className="card mb-4 p-4">
        <h2 className="mb-2 text-sm font-semibold">Your categories</h2>
        {isLoading ? null : custom.length === 0 ? (
          <EmptyState icon={Tags} title="No custom categories" description="Add one for spending the defaults don't cover, like Pet care or Kids." action={<Button size="sm" variant="secondary" onClick={() => setEditing('new')}>Add category</Button>} />
        ) : (
          <ul className="divide-y divide-[var(--border)]">
            {custom.map((c) => (
              <li key={c.id} className="flex items-center gap-3 py-2.5">
                <CategoryIcon category={c} />
                <span className="min-w-0 flex-1 truncate text-sm font-medium">{c.name}</span>
                <button type="button" aria-label={`Edit ${c.name}`} onClick={() => setEditing(c)} className="text-muted rounded-md p-1.5 hover:bg-brand-50 hover:text-brand-700 dark:hover:bg-brand-900/30"><Pencil className="size-4" /></button>
                <button type="button" aria-label={`Delete ${c.name}`} onClick={() => setDeleting(c)} className="text-muted rounded-md p-1.5 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950/40"><Trash2 className="size-4" /></button>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="card p-4">
        <h2 className="mb-2 flex items-center gap-2 text-sm font-semibold">Default categories <Lock className="text-muted size-3.5" /></h2>
        <ul className="grid gap-1 sm:grid-cols-2">
          {system.map((c) => (
            <li key={c.id} className="flex items-center gap-3 py-2">
              <CategoryIcon category={c} />
              <span className="text-sm font-medium">{c.name}</span>
            </li>
          ))}
        </ul>
      </section>

      {editing && <CategoryModal category={editing === 'new' ? null : editing} onClose={() => setEditing(null)} onSaved={() => { setEditing(null); invalidate() }} />}
      <ConfirmDialog open={!!deleting} onClose={() => setDeleting(null)} onConfirm={() => deleting && del.mutate(deleting.id)} loading={del.isPending} title="Delete category" message={`Delete "${deleting?.name ?? ''}"? Receipts in it move to Others.`} />
    </div>
  )
}

function CategoryModal({ category, onClose, onSaved }: { category: Category | null; onClose: () => void; onSaved: () => void }) {
  const [name, setName] = useState(category?.name ?? '')
  const [icon, setIcon] = useState(category?.icon ?? 'tag')
  const [color, setColor] = useState(category?.color ?? '#2a78d6')
  const [error, setError] = useState('')
  const save = useMutation({
    mutationFn: () => {
      if (!name.trim()) throw new Error('Name is required')
      return category ? categoriesApi.update(category.id, { name: name.trim(), icon, color }) : categoriesApi.create({ name: name.trim(), icon, color })
    },
    onSuccess: onSaved,
    onError: (e) => setError(e instanceof ApiRequestError ? Object.values(e.fields ?? {})[0] ?? e.message : (e as Error).message),
  })
  const preview: Category = { id: 'preview', name: name || 'Preview', slug: '', icon, color, isSystem: false }
  return (
    <Modal open onClose={onClose} title={category ? 'Edit category' : 'Add category'} footer={
      <>
        <Button variant="secondary" onClick={onClose}>Cancel</Button>
        <Button onClick={() => save.mutate()} loading={save.isPending}>{category ? 'Save' : 'Add'}</Button>
      </>
    }>
      <form className="grid gap-3" onSubmit={(e) => { e.preventDefault(); save.mutate() }}>
        {error && <Alert>{error}</Alert>}
        <div className="flex items-center gap-3">
          <CategoryIcon category={preview} size="lg" />
          <Field label="Name"><Input name="name" value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Pet care" autoFocus /></Field>
        </div>
        <Field label="Colour">
          <div className="flex items-center gap-2">
            <input type="color" name="color" value={color} onChange={(e) => setColor(e.target.value)} className="h-10 w-14 cursor-pointer rounded-lg border border-[var(--border)] bg-[var(--surface)] p-1" aria-label="Colour" />
            <Input value={color} onChange={(e) => setColor(e.target.value)} className="w-28" aria-label="Colour hex" />
          </div>
        </Field>
        <Field label="Icon">
          <div className="grid grid-cols-6 gap-1.5 sm:grid-cols-8" role="radiogroup" aria-label="Icon">
            {categoryIconNames.map((n) => {
              const I = categoryIcons[n]
              return (
                <button key={n} type="button" role="radio" aria-checked={icon === n} aria-label={n} onClick={() => setIcon(n)} className={clsx('grid h-10 place-items-center rounded-lg border', icon === n ? 'border-brand-600 bg-brand-50 text-brand-700 dark:bg-brand-900/30' : 'border-[var(--border)] hover:border-brand-400')}>
                  <I className="size-4.5" />
                </button>
              )
            })}
          </div>
        </Field>
        <button type="submit" hidden />
      </form>
    </Modal>
  )
}
