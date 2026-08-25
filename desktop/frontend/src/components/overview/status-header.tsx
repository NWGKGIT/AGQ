import { Badge } from '@/components/ui/badge'
import { useCurrentAccount, useStats } from '@/lib/api'
import { ago, maskEmail, until } from '@/lib/format'

/** Current account, freshness, and the nearest quota reset. */
export function StatusHeader({ masked }: { masked: boolean }) {
  const { data: current } = useCurrentAccount()
  const { data: stats } = useStats()

  const display = (email: string) => (masked ? maskEmail(email) : email)

  const accountLive = current?.is_live === true
  const staleSeconds = current?.last_poll_at
    ? (Date.now() - new Date(current.last_poll_at).getTime()) / 1000
    : undefined

  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
      <Badge variant={accountLive ? 'success' : 'secondary'}>
        <span className={`size-1.5 rounded-full ${accountLive ? 'bg-success' : 'bg-muted-foreground'}`} />
        {accountLive ? `Signed in · ${display(current?.email ?? '')}` : 'No account detected'}
      </Badge>
      <div className="ml-auto flex items-center gap-3 text-xs text-muted-foreground">
        {staleSeconds === undefined ? (
          <span>Checking for updates…</span>
        ) : staleSeconds > 10 ? (
          <span className="text-warning">Data may be stale · updated {ago(staleSeconds)} ago</span>
        ) : (
          <span className="tnum">
            Updated {ago(staleSeconds)} ago
          </span>
        )}
        {accountLive && current?.next_poll_at && (
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
      {accountLive && (
        <p className="w-full rounded-md border border-warning/30 bg-warning/10 px-3 py-2 text-xs text-muted-foreground">
          Changed accounts? Restart Antigravity to apply the new login.
        </p>
      )}
    </div>
  )
}
