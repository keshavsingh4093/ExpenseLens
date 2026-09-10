import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, RefreshCw, SlidersHorizontal, Pencil, Trash2 } from 'lucide-react'
import { PageHeader } from '@common/ui/PageHeader'
import { Button } from '@common/ui/Button'
import { Field, Input } from '@common/ui/Field'
import { Modal, ConfirmDialog } from '@common/ui/Modal'
import { Alert } from '@common/ui/Alert'
import { Skeleton } from '@common/ui/Skeleton'
import { EmptyState } from '@common/ui/EmptyState'
import { Toggle } from '@common/ui/Toggle'
import { ApiRequestError } from '@common/api/client'
import { configApi, configKeys, type ConfigEntry } from './api'

const schema = z.object({
  key: z.string().regex(/^[A-Z][A-Z0-9_]{1,63}$/, 'UPPER_SNAKE_CASE, 2–64 chars'),
  value: z.string(),
  description: z.string().max(500).optional(),
  category: z.string().max(40).optional(),
  sensitive: z.boolean(),
})
type Form = z.infer<typeof schema>

export function SystemConfigPage() {
  const qc = useQueryClient()
  const list = useQuery({ queryKey: configKeys.list, queryFn: configApi.list })
  const [editing, setEditing] = useState<ConfigEntry | 'new' | null>(null)
  const [deleting, setDeleting] = useState<ConfigEntry | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const invalidate = () => qc.invalidateQueries({ queryKey: configKeys.list })

  const reload = useMutation({
    mutationFn: configApi.reload,
    onSuccess: (r) => setNotice(r.userBackendReloaded ? 'Reloaded admin and user backends.' : 'Reloaded admin backend (user backend reload not configured).'),
  })
  const remove = useMutation({
    mutationFn: (key: string) => configApi.remove(key),
    onSuccess: () => {
      setDeleting(null)
      invalidate()
    },
  })

  const rows = list.data?.configs ?? []
  const grouped = rows.reduce<Record<string, ConfigEntry[]>>((acc, c) => {
    const k = c.category || 'general'
    ;(acc[k] ??= []).push(c)
    return acc
  }, {})

  return (
    <div>
      <PageHeader
        title="System Config"
        subtitle="Dynamic keys override .env defaults after a reload"
        actions={
          <>
            <Button variant="secondary" onClick={() => reload.mutate()} loading={reload.isPending}>
              <RefreshCw className="size-4" /> Reload
            </Button>
            <Button onClick={() => setEditing('new')}>
              <Plus className="size-4" /> Add key
            </Button>
          </>
        }
      />
      {notice && <div className="mb-4"><Alert kind="success">{notice}</Alert></div>}
      {list.error && <Alert>Could not load configuration.</Alert>}
      {list.isLoading ? (
        <div className="grid gap-2">{Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-12" />)}</div>
      ) : rows.length === 0 ? (
        <div className="card">
          <EmptyState icon={SlidersHorizontal} title="No dynamic config yet" description="Keys added here are read by the backends on reload." action={<Button onClick={() => setEditing('new')}>Add key</Button>} />
        </div>
      ) : (
        Object.entries(grouped).map(([cat, items]) => (
          <section key={cat} className="card mb-4 overflow-x-auto">
            <h2 className="border-b border-[var(--border)] px-4 py-2 text-xs font-semibold tracking-wide text-muted uppercase">{cat}</h2>
            <table className="w-full min-w-[560px] text-sm">
              <tbody className="divide-y divide-[var(--border)]">
                {items.map((c) => (
                  <tr key={c.key}>
                    <td className="px-4 py-3 align-top">
                      <div className="font-mono text-xs font-semibold">{c.key}</div>
                      {c.description && <div className="text-muted mt-0.5 text-xs">{c.description}</div>}
                    </td>
                    <td className="max-w-[320px] truncate px-4 py-3 font-mono text-xs" title={c.sensitive ? 'masked' : c.value}>
                      {c.value || <span className="text-muted">(empty)</span>}
                    </td>
                    <td className="px-4 py-3 text-right whitespace-nowrap">
                      <button type="button" aria-label={`Edit ${c.key}`} onClick={() => setEditing(c)} className="text-muted rounded-md p-1.5 hover:bg-brand-50 hover:text-brand-700 dark:hover:bg-brand-900/30">
                        <Pencil className="size-4" />
                      </button>
                      <button type="button" aria-label={`Delete ${c.key}`} onClick={() => setDeleting(c)} className="text-muted rounded-md p-1.5 hover:bg-red-50 hover:text-red-700 dark:hover:bg-red-950/40">
                        <Trash2 className="size-4" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>
        ))
      )}
      <EditDialog entry={editing} onClose={() => setEditing(null)} onSaved={() => { setEditing(null); invalidate() }} />
      <ConfirmDialog
        open={!!deleting}
        onClose={() => setDeleting(null)}
        onConfirm={() => deleting && remove.mutate(deleting.key)}
        title="Delete key"
        message={`Remove ${deleting?.key ?? ''}? Backends fall back to the .env value or code default.`}
        loading={remove.isPending}
      />
    </div>
  )
}

function EditDialog({ entry, onClose, onSaved }: { entry: ConfigEntry | 'new' | null; onClose: () => void; onSaved: () => void }) {
  const isNew = entry === 'new'
  const current = entry && entry !== 'new' ? entry : null
  const { register, handleSubmit, setError, watch, setValue, formState: { errors, isSubmitting } } = useForm<Form>({
    resolver: zodResolver(schema),
    values: {
      key: current?.key ?? '',
      value: current?.value ?? '',
      description: current?.description ?? '',
      category: current?.category ?? '',
      sensitive: current?.sensitive ?? false,
    },
  })
  const sensitive = watch('sensitive')
  const onSubmit = async (v: Form) => {
    try {
      await configApi.upsert(v.key, { value: v.value, description: v.description, category: v.category, sensitive: v.sensitive })
      onSaved()
    } catch (e) {
      if (e instanceof ApiRequestError && e.fields) {
        for (const [k, msg] of Object.entries(e.fields)) setError(k as keyof Form, { message: msg })
      } else setError('root', { message: e instanceof ApiRequestError ? e.message : 'Could not save' })
    }
  }
  return (
    <Modal open={!!entry} onClose={onClose} title={isNew ? 'Add key' : `Edit ${current?.key}`}>
      <form onSubmit={handleSubmit(onSubmit)} className="grid gap-3" noValidate>
        {errors.root && <Alert>{errors.root.message}</Alert>}
        <Field label="Key" error={errors.key?.message} hint="UPPER_SNAKE_CASE, e.g. GEMINI_MODEL">
          <Input {...register('key')} disabled={!isNew} invalid={!!errors.key} className="font-mono" placeholder="GEMINI_MODEL" />
        </Field>
        <Field label="Value" error={errors.value?.message} hint={sensitive && current ? 'Leave the masked value unchanged to keep the stored secret.' : undefined}>
          <Input {...register('value')} className="font-mono" type={sensitive ? 'password' : 'text'} autoComplete="off" />
        </Field>
        <Field label="Description" error={errors.description?.message}>
          <Input {...register('description')} />
        </Field>
        <Field label="Category" error={errors.category?.message} hint="Groups keys in this list, e.g. extraction, storage, mail">
          <Input {...register('category')} placeholder="general" />
        </Field>
        <Toggle checked={sensitive} onChange={(v) => setValue('sensitive', v)} label="Sensitive" description="Mask the value in listings" />
        <div className="mt-2 flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>Cancel</Button>
          <Button type="submit" loading={isSubmitting}>Save</Button>
        </div>
      </form>
    </Modal>
  )
}
