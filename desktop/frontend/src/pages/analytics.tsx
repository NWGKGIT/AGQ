import { useState } from 'react'

import { BreakdownTable } from '@/components/analytics/breakdown-table'
import { StatCards } from '@/components/analytics/stat-cards'
import { UsageChart, type AggKey, type RangeKey } from '@/components/analytics/usage-chart'
import { DaemonUnreachable } from '@/components/daemon-unreachable'
import { isDaemonUnreachable, useAppConfig, useStats } from '@/lib/api'

export function AnalyticsPage() {
  const { data: cfg } = useAppConfig()
  const { error } = useStats()
  const [range, setRange] = useState<RangeKey>('7d')
  const [agg, setAgg] = useState<AggKey>('avg')

  const masked = cfg?.mask_emails ?? false

  if (error && isDaemonUnreachable(error)) {
    return <DaemonUnreachable />
  }

  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-5 p-6">
      <header className="flex items-end justify-between border-b pb-4">
        <div>
          <h1 className="text-lg font-semibold">Analytics</h1>
          <p className="mt-0.5 text-sm text-muted-foreground">
            Usage trends and consumption across accounts.
          </p>
        </div>
      </header>

      <StatCards masked={masked} />
      <UsageChart range={range} agg={agg} onRangeChange={setRange} onAggChange={setAgg} />
      <BreakdownTable masked={masked} />
    </div>
  )
}
