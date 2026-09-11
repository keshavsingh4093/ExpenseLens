import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Lock, Pencil, Plus, Tags, Trash2 } from 'lucide-react'
import { clsx } from 'clsx'
import { PageHeader } from '@common/ui/PageHeader'
import { Button } from '@common/ui/Button'
import { EmptyState } from '@common/ui/EmptyState'
import { Modal, ConfirmDialog } from '@common/ui/Modal'
import { Field, Input } from '@common/ui/Field'
import { Alert } from '@common/ui/Alert'
import { Skeleton } from '@common/ui/Skeleton'
import { CategoryIcon } from '@common/categories/CategoryIcon'
import { categoryIcons, categoryIconNames } from '@common/categories/icons'
import type { Category } from '@common/categories/types'
import { ApiRequestError } from '@common/api/client'
import { categoriesApi, categoryKeys, type SystemCategory } from './api'

const OTHERS = 'others'

export function CategoriesPage() {
  const qc = useQueryClient()
  const list = useQuery({ queryKey: categoryKeys.list, queryFn: categoriesApi.list })
  const [editing, setEditing] = useState<SystemCategory | 'new' | null>(null)
  const [deleting, setDeleting] = useState<SystemCategory | null>(null)
  const [notice, setNotice] = useState('')
  const invalidate = () => qc.invalidateQueries({ queryKey: categoryKeys.list })
  const del = useMutation({
    mutationFn: (id: string) => categoriesApi.remove(id),
    onSuccess: (r) => {
      setNotice(`Deleted "${deleting?.name}". ${r.receiptsMoved} receipt${r.receiptsMoved === 1 ? '' : 's'} moved to Others.`)
      setDeleting(null)
      invalidate()
    },
  })
  const rows = list.data?.categories ?? []

  return (
    <div className="mx-auto max-w-4xl">
      <PageHeader
        title="Categories"
        subtitle="Default categories every user sees. Extraction suggests them by slug."
        actions={<Button onClick={() => { setNotice(''); setEditing('new') }}><Plus className="size-4" /> Add category</Button>}
      />
      {notice && <div className="mb-4"><Alert kind="success">{notice}</Alert></div>}
      {list.error && <Alert>Could not load categories.</Alert>}
      <section className="card">
        {list.isLoading ? (
          <div className="grid gap-2 p-4">{Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} className="h-12" />)}</div>
        ) : rows.length === 0 ? (
          <EmptyState icon={Tags} title="No system categories" description="Run make ensure-indexes to seed the defaults, or add one." />
        ) : (
          <ul className="divide-y divide-[var(--border)]">
            {rows.map((c) => (
              <li key={c.id} className="flex items-center gap-3 px-4 py-3">
                <CategoryIcon category={c} />
                <div className="min-w-0 flex-1">
                  <div className="truncate text-sm font-medium">{c.name}</div>
                  <div className="text-muted truncate font-mono text-xs">{c.slug}</div>
                </div>
                <span className="text-muted text-xs tabular-nums whitespace-nowrap">
                  {c.receiptCount.toLocaleString('en-IN')} receipt{c.receiptCount === 1 ? '' : 's'}
                </span>
                <button type="button" aria-label={`Edit ${c.name}`} onClick={() => { setNotice(''); setEditing(c) }} className="text-muted rounded-md p-1.5 hover:bg-brand-50 hover:text-brand-700 dark:hover:bg-brand-900/30">
                  <Pencil className="size-4" />
                </button>
                {c.slug === OTHERS ? (
                  <span className="text-muted p-1.5" title="Others is the fallback category and cannot be deleted" aria-label="Others cannot be deleted">
                    <Lock className="size-4" />
                  </span>
                ) : (
                  <button type="button" aria-label={`Delete ${c.name}`} onClick={() => { setNotice(''); setDeleting(c) }} className="text-muted rounded-md p-1.5 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950/40">
                    <Trash2 className="size-4" />
                  </button>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>

      {editing && (
        <CategoryModal
          category={editing === 'new' ? null : editing}
          onClose={() => setEditing(null)}
          onSaved={(msg) => { setEditing(null); setNotice(msg); invalidate() }}
        />
      )}
      <ConfirmDialog
        open={!!deleting}
        onClose={() => setDeleting(null)}
        onConfirm={() => deleting && del.mutate(deleting.id)}
        loading={del.isPending}
        title="Delete category"
        message={`Delete "${deleting?.name ?? ''}" for every user? ${deleting?.receiptCount ?? 0} receipt${deleting?.receiptCount === 1 ? '' : 's'} will move to Others.`}
      />
    </div>
  )
}

function CategoryModal({ category, onClose, onSaved }: { category: SystemCategory | null; onClose: () => void; onSaved: (msg: string) => void }) {
  const [name, setName] = useState(category?.name ?? '')
  const [slug, setSlug] = useState(category?.slug ?? '')
  const [icon, setIcon] = useState(category?.icon ?? 'tag')
  const [color, setColor] = useState(category?.color ?? '#2a78d6')
  const [error, setError] = useState('')
  const isOthers = category?.slug === OTHERS
  const save = useMutation({
    mutationFn: () => {
      if (!name.trim()) throw new Error('Name is required')
      const body = { name: name.trim(), icon, color, ...(slug.trim() && !isOthers ? { slug: slug.trim() } : {}) }
      return category ? categoriesApi.update(category.id, body) : categoriesApi.create(body)
    },
    onSuccess: (c) => onSaved(category ? `Saved "${c.name}".` : `Added "${c.name}" (${c.slug}).`),
    onError: (e) => setError(e instanceof ApiRequestError ? Object.entries(e.fields ?? {}).map(([k, v]) => `${k} ${v}`)[0] ?? e.message : (e as Error).message),
  })
  const preview: Category = { id: 'preview', name: name || 'Preview', slug, icon, color, isSystem: true }
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
          <div className="min-w-0 flex-1">
            <Field label="Name"><Input name="name" value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Pet care" autoFocus /></Field>
          </div>
        </div>
        <Field label="Slug" hint={isOthers ? 'The fallback slug cannot change.' : category ? 'Changing a slug affects extraction suggestions.' : 'Optional. Derived from the name when empty.'}>
          <Input name="slug" value={slug} onChange={(e) => setSlug(e.target.value)} disabled={isOthers} className="font-mono" placeholder="pet-care" />
        </Field>
        <Field label="Colour">
          <div className="flex items-center gap-2">
            <input type="color" value={color} onChange={(e) => setColor(e.target.value)} className="h-10 w-14 cursor-pointer rounded-lg border border-[var(--border)] bg-[var(--surface)] p-1" aria-label="Colour" />
            <Input value={color} onChange={(e) => setColor(e.target.value)} className="w-28" aria-label="Colour hex" />
          </div>
        </Field>
        <div>
          <span className="mb-1 block text-sm font-medium" id="icon-label">Icon</span>
          <div className="grid grid-cols-6 gap-1.5 sm:grid-cols-8" role="radiogroup" aria-labelledby="icon-label">
            {categoryIconNames.map((n) => {
              const I = categoryIcons[n]
              return (
                <button key={n} type="button" role="radio" aria-checked={icon === n} aria-label={n} onClick={() => setIcon(n)} className={clsx('grid h-10 place-items-center rounded-lg border', icon === n ? 'border-brand-600 bg-brand-50 text-brand-700 dark:bg-brand-900/30' : 'border-[var(--border)] hover:border-brand-400')}>
                  <I className="size-4.5" />
                </button>
              )
            })}
          </div>
        </div>
        <button type="submit" hidden />
      </form>
    </Modal>
  )
}
