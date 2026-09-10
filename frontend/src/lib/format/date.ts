/** Formats a YYYY-MM-DD string (or ISO datetime) as "12 Jun 2026". */
export function formatDate(iso: string, locale = 'en-IN'): string {
  const d = iso.length === 10 ? new Date(`${iso}T00:00:00`) : new Date(iso)
  return new Intl.DateTimeFormat(locale, { day: '2-digit', month: 'short', year: 'numeric' }).format(d)
}
