import { Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import type { TrendPoint } from '../analytics/types'
import { formatMoney } from '../format/money'

const compact = (n: number) => {
  const r = n / 100
  if (r >= 1_00_000) return `₹${(r / 1_00_000).toFixed(1)}L`
  if (r >= 1000) return `₹${(r / 1000).toFixed(r >= 10_000 ? 0 : 1)}K`
  return `₹${r.toFixed(0)}`
}

/** Single-series area/line trend. One series, so no legend: the card title names it. */
export function TrendChart({ points, currency = 'INR', height = 220 }: { points: TrendPoint[]; currency?: string; height?: number }) {
  const every = Math.max(1, Math.ceil(points.length / 7))
  return (
    <div style={{ height }} className="w-full">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={points} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
          <defs>
            <linearGradient id="trendFill" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="var(--color-brand-600)" stopOpacity={0.14} />
              <stop offset="100%" stopColor="var(--color-brand-600)" stopOpacity={0.02} />
            </linearGradient>
          </defs>
          <CartesianGrid vertical={false} stroke="var(--border)" strokeWidth={1} />
          <XAxis dataKey="label" tickLine={false} axisLine={false} interval={every - 1} tick={{ fontSize: 11, fill: 'var(--text-muted)' }} dy={6} />
          <YAxis tickLine={false} axisLine={false} width={44} tickFormatter={compact} tick={{ fontSize: 11, fill: 'var(--text-muted)' }} />
          <Tooltip
            cursor={{ stroke: 'var(--text-muted)', strokeWidth: 1 }}
            content={({ active, payload }) => {
              if (!active || !payload?.length) return null
              const p = payload[0].payload as TrendPoint
              return (
                <div className="card px-3 py-2 text-xs shadow-md">
                  <div className="text-muted">{p.label}</div>
                  <div className="font-semibold">{formatMoney(p.total, currency)}</div>
                  <div className="text-muted">{p.count} receipt{p.count === 1 ? '' : 's'}</div>
                </div>
              )
            }}
          />
          <Area
            type="monotone"
            dataKey="total"
            stroke="var(--color-brand-600)"
            strokeWidth={2}
            fill="url(#trendFill)"
            dot={false}
            activeDot={{ r: 5, strokeWidth: 2, stroke: 'var(--surface)', fill: 'var(--color-brand-600)' }}
            isAnimationActive={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}
