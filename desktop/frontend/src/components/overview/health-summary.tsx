import { AlertTriangle, CircleHelp, CircleCheck, OctagonAlert } from 'lucide-react'

import { Card } from '@/components/ui/card'
import { deriveHealth, HEALTH_LABELS, type HealthStatus } from '@/lib/health'
import { cn } from '@/lib/utils'
import type { apiclient } from '../../../wailsjs/go/models'

const SUMMARY_ITEMS: Array<{
  status: HealthStatus
  icon: typeof OctagonAlert
  className: string
}> = [
  { status: 'low', icon: OctagonAlert, className: 'text-destructive' },
  { status: 'warning', icon: AlertTriangle, className: 'text-warning' },
  { status: 'unknown', icon: CircleHelp, className: 'text-muted-foreground' },
  { status: 'good', icon: CircleCheck, className: 'text-success' },
]

export function HealthSummary({ accounts }: { accounts: readonly apiclient.Account[] }) {
  const counts: Record<HealthStatus, number> = { low: 0, warning: 0, unknown: 0, good: 0 }
  for (const account of accounts) {
    counts[deriveHealth(account.latest_snapshot?.models ?? []).status] += 1
  }

  return (
    <Card className="grid grid-cols-2 sm:grid-cols-4 sm:divide-x">
      {SUMMARY_ITEMS.map(({ status, icon: Icon, className }) => (
        <div key={status} className="flex items-center gap-3 px-4 py-3">
          <Icon className={cn('size-4 shrink-0', className)} aria-hidden="true" />
          <div>
            <div className="font-mono text-xl font-semibold leading-none">{counts[status]}</div>
            <div className="mt-1 text-xs text-muted-foreground">{HEALTH_LABELS[status]}</div>
          </div>
        </div>
      ))}
    </Card>
  )
}
