import { downloadFile } from '@common/api/client'
import type { RangeParams } from '@common/analytics/types'

const tz = () => Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Kolkata'

export type ReportFormat = 'csv' | 'pdf'

export const reportsApi = {
  export: (format: ReportFormat, r: RangeParams) => {
    const p = new URLSearchParams({ format, tz: tz() })
    for (const [k, v] of Object.entries(r)) if (v) p.set(k, String(v))
    return downloadFile(`/api/reports/export?${p.toString()}`, `expenselens.${format}`)
  },
  exportAll: () => downloadFile('/api/me/export', 'expenselens-export.json'),
}
