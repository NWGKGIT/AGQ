import { Badge } from '@/components/ui/badge'
import { useCurrentAccount, useStats } from '@/lib/api'
import { ago, maskEmail, until } from '@/lib/format'

/**
 * Monitor liveness line: state badge, active (or last known) account, poll
 * cadence, and the soonest upcoming reset across all accounts.
 */
export function StatusHeader({ masked }: { masked: boolean }) {
  const { data: current } = useCurrentAccount()
  const { data: stats } = useStats()

  const display = (email: string) => (masked ? maskEmail(email) : email)

  const active = current?.is_live
  const email = current?.email || current?.last_account?.email

  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
      <Badge variant={active ? 'success' : 'secondary'}>
        <span
          className={`size-1.5 rounded-full ${active ? 'bg-success' : 'bg-muted-foreground'}`}
        />
        {current?.state ?? '…'}
      </Badge>
      {email && (
        <span className="font-mono text-sm">
          {display(email)}
          {!active && <span className="ml-2 text-xs text-muted-foreground">last seen</span>}
          <span className="ml-2 text-xs text-muted-foreground">
            (<a href="#" onClick={(e) => e.preventDefault()} title="Restart Antigravity if the account looks wrong" className="underline underline-offset-1 hover:text-foreground">restart for refresh</a>)
          </span>
        </span>
      )}
      <div className="ml-auto flex items-center gap-3 text-xs text-muted-foreground">
        {current?.last_poll_at && (
          <span className="tnum">
            polled{' '}
            {ago((Date.now() - new Date(current.last_poll_at).getTime()) / 1000)} ago
          </span>
        )}
        {active && current?.next_poll_at && (
          <span className="tnum">next in {until(current.next_poll_at)}</span>
        )}
        {stats?.next_reset && (
          <Badge variant="outline" className="gap-1.5 font-normal text-muted-foreground">
            next reset
            <span className="tnum font-medium text-foreground">
              {until(stats.next_reset.reset_time)}
            </span>
            <span className="max-w-40 truncate">{display(stats.next_reset.email)}</span>
          </Badge>
        )}
      </div>
    </div>
  )
}
