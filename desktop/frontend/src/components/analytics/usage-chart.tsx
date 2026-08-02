import { useMemo } from 'react'
import {
  Area,
  AreaChart,
  CartesianGrid,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useTimeseries } from '@/lib/api'
import { PROVIDERS, PROVIDER_COLORS } from '@/lib/providers'
import { cn } from '@/lib/utils'

export type RangeKey = '7d' | '30d'
export type AggKey = 'avg' | 'min'

function Toggle<T extends string>({
  value,
  options,
  onChange,
}: {
  value: T
  options: readonly T[]
  onChange: (v: T) => void
}) {
  return (
    <div className="flex overflow-hidden rounded-md border text-xs">
      {options.map((option) => (
        <button
          key={option}
          onClick={() => onChange(option)}
          className={cn(
            'px-2.5 py-1 transition-colors duration-150',
            option === value
              ? 'bg-primary text-primary-foreground'
              : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground',
          )}
        >
          {option}
        </button>
      ))}
    </div>
  )
}

/**
 * Per-day remaining-quota trend, one gradient-filled line per provider.
 * Lines read trends better than the previous grouped bars: quota is a
 * continuous level, not a per-day quantity.
 */
export function UsageChart({
  range,
  agg,
  onRangeChange,
  onAggChange,
}: {
  range: RangeKey
  agg: AggKey
  onRangeChange: (r: RangeKey) => void
  onAggChange: (a: AggKey) => void
}) {
  const { data, isPending } = useTimeseries(range, agg)

  const chartData = useMemo(
    () =>
      (data?.days ?? []).map((day) => ({
        // "2026-07-01" -> "Jul 1"; keep it short on the axis.
        date: new Date(`${day.date}T00:00:00Z`).toLocaleDateString(undefined, {
          month: 'short',
          day: 'numeric',
          timeZone: 'UTC',
        }),
        Gemini: day.providers['Gemini'] != null ? day.providers['Gemini'] * 100 : null,
        Anthropic: day.providers['Anthropic'] != null ? day.providers['Anthropic'] * 100 : null,
        OpenAI: day.providers['OpenAI'] != null ? day.providers['OpenAI'] * 100 : null,
      })),
    [data],
  )

  return (
    <Card className="flex h-[320px] flex-col p-4">
      <div className="mb-3 flex items-center justify-between">
        <h2 className="text-xs font-medium uppercase tracking-widest text-muted-foreground">
          Remaining quota over time
        </h2>
        <div className="flex gap-2">
          <Toggle value={agg} options={['avg', 'min'] as const} onChange={onAggChange} />
          <Toggle value={range} options={['7d', '30d'] as const} onChange={onRangeChange} />
        </div>
      </div>
      {isPending ? (
        <Skeleton className="flex-1" />
      ) : chartData.length === 0 ? (
        <div className="flex flex-1 items-center justify-center text-sm text-muted-foreground">
          No samples in this range yet.
        </div>
      ) : (
        <ResponsiveContainer width="100%" height="100%" className="flex-1">
          <AreaChart data={chartData} margin={{ top: 4, right: 4, bottom: 0, left: -24 }}>
            <defs>
              {PROVIDERS.map((provider) => (
                <linearGradient
                  key={provider}
                  id={`fill-${provider}`}
                  x1="0"
                  y1="0"
                  x2="0"
                  y2="1"
                >
                  <stop offset="0%" stopColor={PROVIDER_COLORS[provider]} stopOpacity={0.18} />
                  <stop offset="100%" stopColor={PROVIDER_COLORS[provider]} stopOpacity={0.02} />
                </linearGradient>
              ))}
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} />
            <XAxis
              dataKey="date"
              tick={{ fontSize: 11, fill: 'var(--muted-foreground)' }}
              tickLine={false}
              axisLine={{ stroke: 'var(--border)' }}
            />
            <YAxis
              domain={[0, 100]}
              unit="%"
              tick={{ fontSize: 11, fill: 'var(--muted-foreground)' }}
              tickLine={false}
              axisLine={false}
            />
            <Tooltip
              formatter={(value) => [`${Math.round(Number(value))}%`]}
              cursor={{ stroke: 'var(--muted-foreground)', strokeDasharray: '3 3' }}
              contentStyle={{
                backgroundColor: 'var(--popover)',
                border: '1px solid var(--border)',
                borderRadius: 8,
                fontSize: 12,
                color: 'var(--popover-foreground)',
              }}
            />
            <Legend wrapperStyle={{ fontSize: 12 }} iconType="circle" iconSize={8} />
            {PROVIDERS.map((provider) => (
              <Area
                key={provider}
                type="monotone"
                dataKey={provider}
                stroke={PROVIDER_COLORS[provider]}
                strokeWidth={2}
                fill={`url(#fill-${provider})`}
                dot={false}
                activeDot={{ r: 3.5, strokeWidth: 0 }}
                connectNulls
              />
            ))}
          </AreaChart>
        </ResponsiveContainer>
      )}
    </Card>
  )
}
