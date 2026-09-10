/** Formats a YYYY-MM-DD string (or ISO datetime) as "12 Jun 2026". */
export function formatDate(iso: string, locale = 'en-IN'): string {
  if (!iso) return ''
  const d = iso.length === 10 ? new Date(`${iso}T00:00:00`) : new Date(iso)
  return new Intl.DateTimeFormat(locale, { day: '2-digit', month: 'short', year: 'numeric' }).format(d)
}

/** "12 Jun 2026, 10:45 AM" from an ISO datetime, or date + HH:MM. */
export function formatDateTime(date: string, time?: string, locale = 'en-IN'): string {
  const base = formatDate(date, locale)
  if (!time) return base
  const [h, m] = time.split(':').map(Number)
  if (Number.isNaN(h)) return base
  const d = new Date(2000, 0, 1, h, m || 0)
  return `${base}, ${new Intl.DateTimeFormat(locale, { hour: 'numeric', minute: '2-digit' }).format(d)}`
}

/** Today's date as YYYY-MM-DD in local time. */
export function todayISO(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

/** "2.4 MB" style file sizes. */
export function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(0)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}
