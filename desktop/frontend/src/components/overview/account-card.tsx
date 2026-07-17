import { QuotaBar } from '@/components/quota-bar'
import { Card } from '@/components/ui/card'
import { ago, maskEmail, pct, until } from '@/lib/format'
import { groupByProvider } from '@/lib/providers'
import { cn } from '@/lib/utils'
import type { apiclient } from '../../../wailsjs/go/models'

const MODELS_SHOWN_PER_PROVIDER = 3

function soonestReset(models: apiclient.ModelQuota[]): string | undefined {
  const times = models
    .map((m) => m.reset_time)
    .filter((t): t is string => !!t)
    .sort()
  return times[0]
}

/**
 * One account with its per-provider model quotas. Clicking opens the detail
 * sheet. Fully exhausted accounts are dimmed like the design samples.
 */
export function AccountCard({
  account,
  live,
  masked,
  onSelect,
}: {
  account: apiclient.Account
  live: boolean
  masked: boolean
  onSelect: (email: string) => void
}) {
  const snapshot = account.latest_snapshot
  const models = snapshot?.models ?? []
  const exhausted =
    models.length > 0 && models.every((m) => m.is_exhausted || m.remaining_fraction === 0)
  const groups = groupByProvider(models, (m) => m.label)
  const nextReset = soonestReset(models)

  return (
    <Card
      role="button"
      tabIndex={0}
      onClick={() => onSelect(account.email)}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') onSelect(account.email)
      }}
      className={cn(
        'flex cursor-pointer flex-col gap-3 p-4 transition-colors hover:border-ring/60',
        exhausted && 'opacity-50',
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="truncate font-mono text-sm">
          {masked ? maskEmail(account.email) : account.email}
        </span>
        <span
          className={cn(
            'size-2 shrink-0 rounded-full',
            live ? 'bg-success' : exhausted ? 'bg-destructive/60' : 'bg-muted-foreground/40',
          )}
          title={live ? 'live' : exhausted ? 'exhausted' : 'idle'}
        />
      </div>

      <div className="flex justify-between text-[11px] uppercase tracking-wide text-muted-foreground">
        <span>{snapshot ? `synced ${ago(snapshot.staleness_seconds)} ago` : 'no snapshot'}</span>
        <span>{nextReset ? `resets ${until(nextReset)}` : account.plan_name || ''}</span>
      </div>

      <div className="h-px bg-border" />

      {groups.length === 0 ? (
        <p className="py-2 text-xs text-muted-foreground">No model quota data yet.</p>
      ) : (
        <div className="flex flex-col gap-3">
          {groups.map(({ provider, items }) => {
            const shown = items.slice(0, MODELS_SHOWN_PER_PROVIDER)
            const hidden = items.length - shown.length
            return (
              <div key={provider}>
                <span className="mb-1.5 block text-[10px] font-semibold uppercase tracking-widest text-muted-foreground/70">
                  {provider}
                </span>
                <div className="flex flex-col gap-1.5">
                  {shown.map((m) => (
                    <div key={m.model_id + m.label} className="flex items-center gap-2 text-xs">
                      <span
                        className="w-24 truncate text-muted-foreground"
                        title={m.label}
                      >
                        {m.label}
                      </span>
                      <QuotaBar fraction={m.remaining_fraction} className="h-[2px] flex-1" />
                      <span className="w-9 text-right font-mono">
                        {pct(m.remaining_fraction)}
                      </span>
                      <span className="w-12 text-right font-mono text-muted-foreground">
                        {m.reset_time ? until(m.reset_time) : '–'}
                      </span>
                    </div>
                  ))}
                  {hidden > 0 && (
                    <span className="text-[10px] text-muted-foreground">+{hidden} more</span>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      )}
    </Card>
  )
}
