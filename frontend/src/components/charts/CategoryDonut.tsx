import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts'
import { formatMoney } from '@/lib/format/money'
import type { DonutSlice } from './donutSlices'

export function CategoryDonut({ slices, total, currency = 'INR', size = 180 }: { slices: DonutSlice[]; total: number; currency?: string; size?: number }) {
  return (
    <div className="flex flex-wrap items-center gap-4">
      <div style={{ width: size, height: size }} className="relative shrink-0">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={slices}
              dataKey="total"
              nameKey="name"
              innerRadius="68%"
              outerRadius="100%"
              paddingAngle={2}
              stroke="var(--surface)"
              strokeWidth={2}
              isAnimationActive={false}
            >
              {slices.map((s) => <Cell key={s.id} fill={s.color} />)}
            </Pie>
            <Tooltip
              content={({ active, payload }) => {
                if (!active || !payload?.length) return null
                const s = payload[0].payload as DonutSlice
                return (
                  <div className="card px-3 py-2 text-xs shadow-md">
                    <div className="flex items-center gap-1.5"><span className="size-2.5 rounded-sm" style={{ background: s.color }} /> {s.name}</div>
                    <div className="font-semibold">{formatMoney(s.total, currency)} · {s.percent.toFixed(0)}%</div>
                  </div>
                )
              }}
            />
          </PieChart>
        </ResponsiveContainer>
        <div className="pointer-events-none absolute inset-0 grid place-items-center text-center">
          <div>
            <div className="text-lg font-semibold">{formatMoney(total, currency)}</div>
            <div className="text-muted text-[11px]">Total</div>
          </div>
        </div>
      </div>
      <ul className="grid min-w-0 flex-1 gap-1.5 text-sm">
        {slices.map((s) => (
          <li key={s.id} className="flex items-center gap-2">
            <span className="size-2.5 shrink-0 rounded-sm" style={{ background: s.color }} />
            <span className="truncate">{s.name}</span>
            <span className="text-muted ml-auto tabular-nums">{s.percent.toFixed(0)}%</span>
          </li>
        ))}
      </ul>
    </div>
  )
}
