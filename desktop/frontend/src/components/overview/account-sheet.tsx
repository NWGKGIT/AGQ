import { Skeleton } from '@/components/ui/skeleton'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useAccountModels, useSnapshots, useTimeline } from '@/lib/api'
import { ago, maskEmail, pct, shortDateTime, timeOnly, until } from '@/lib/format'
import { healthStatusForFraction } from '@/lib/health'
import { PROVIDERS, PROVIDER_COLORS, groupByProvider, type Provider } from '@/lib/providers'
import { cn } from '@/lib/utils'
import type { apiclient } from '../../../wailsjs/go/models'

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <h3 className="mb-3 text-xs font-medium uppercase tracking-widest text-muted-foreground">
      {children}
    </h3>
  )
}

/** A model's remaining quota as a provider-colored bar with a bold readout. */
function ModelRow({ model }: { model: apiclient.ModelAggregate }) {
  const fraction = model.remaining_fraction ?? null
  const low = healthStatusForFraction(fraction) === 'low'
  const provider = groupByProvider([model], (m) => m.label)[0]?.provider
  const barColor = low
    ? 'var(--destructive)'
    : provider
      ? PROVIDER_COLORS[provider]
      : 'var(--muted-foreground)'
  const reset = model.pool_reset_time ?? model.reset_time
  const width = fraction != null ? Math.max(0, Math.min(1, fraction)) * 100 : 0

  return (
    <div className="interactive-surface group rounded-md border border-transparent px-2 py-2">
      <div className="flex items-baseline justify-between gap-3">
        <span className="truncate text-sm" title={model.label}>
          {model.label}
        </span>
        <span className="flex shrink-0 items-baseline gap-2">
          {model.assumed_refilled && (
            <span
              className="rounded-full bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground"
              title="The reset time passed with no fresh poll; quota is assumed refilled."
            >
              assumed
            </span>
          )}
          <span className={cn('tnum font-mono text-sm font-semibold', low && 'text-destructive')}>
            {pct(fraction)}
          </span>
        </span>
      </div>
      <div
        className="mt-1.5 h-1.5 w-full overflow-hidden rounded-full bg-muted"
        role="progressbar"
        aria-label={`${model.label} quota remaining`}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={fraction != null ? Math.round(width) : undefined}
      >
        <div
          className="h-full rounded-full transition-all duration-300"
          style={{ width: `${width}%`, backgroundColor: barColor }}
        />
      </div>
      <div className="mt-1 flex justify-between font-mono text-[11px] text-muted-foreground">
        <span>{reset ? `resets in ${until(reset)}` : 'no reset scheduled'}</span>
        <span title={shortDateTime(model.captured_at)}>
          polled {ago(model.staleness_seconds)} ago
        </span>
      </div>
    </div>
  )
}

/**
 * Per-model remaining quotas grouped by provider. Fed by /models/current, so
 * every row carries the newest non-null value — never a blank dash.
 */
function ModelQuotas({ email }: { email: string }) {
  const { data, isPending } = useAccountModels(email)
  if (isPending) return <Skeleton className="h-40 w-full" />
  const models = data?.models ?? []
  const groups = groupByProvider(models, (m) => m.label)
  if (groups.length === 0) {
    return (
      <p className="text-xs text-muted-foreground">
        No quota data yet — model percentages appear after the next poll.
      </p>
    )
  }
  return (
    <div className="flex flex-col gap-5">
      {groups.map(({ provider, items }) => {
        const fractions = items
          .map((m) => m.remaining_fraction)
          .filter((f): f is number => f != null)
        const avg =
          fractions.length > 0 ? fractions.reduce((a, b) => a + b, 0) / fractions.length : null
        const reset = items
          .map((m) => m.pool_reset_time ?? m.reset_time)
          .filter((t): t is string => !!t)
          .sort()[0]
        return (
          <section key={provider}>
            <div className="mb-2 flex items-baseline justify-between">
              <span className="flex items-center gap-1.5 text-xs font-semibold uppercase tracking-widest text-muted-foreground">
                <span
                  className="size-2 rounded-full"
                  style={{ backgroundColor: PROVIDER_COLORS[provider as Provider] }}
                  aria-hidden="true"
                />
                {provider}
              </span>
              <span className="font-mono text-[11px] text-muted-foreground">
                <span className="tnum font-semibold text-foreground">{pct(avg)}</span>
                {reset ? ` · resets in ${until(reset)}` : ''}
              </span>
            </div>
            <div className="flex flex-col gap-1">
              {items.map((m) => (
                <ModelRow key={m.model_id + m.label} model={m} />
              ))}
            </div>
          </section>
        )
      })}
    </div>
  )
}

function Timeline({ email }: { email: string }) {
  const { data, isPending } = useTimeline(email)
  if (isPending) return <Skeleton className="h-24 w-full" />
  const events = [...(data?.events ?? [])].reverse() // newest first
  if (events.length === 0) {
    return <p className="text-xs text-muted-foreground">No sessions in the last 7 days.</p>
  }
  return (
    <div className="relative ml-1 flex flex-col border-l">
      {events.map((e, i) => (
        <div
          key={`${e.at}-${i}`}
          className="interactive-surface relative rounded-md border border-transparent py-2 pl-4 pr-2"
        >
          <span
            className={cn(
              'absolute -left-[4.5px] top-3.5 size-2 rounded-full border-2 border-background',
              e.type === 'login' ? 'bg-success' : 'bg-muted-foreground',
            )}
          />
          <div className="flex items-baseline justify-between gap-2">
            <span className="text-sm capitalize">
              {e.type === 'login' ? 'Logged in' : 'Logged out'}
            </span>
            <span className="tnum font-mono text-[11px] text-muted-foreground">
              {shortDateTime(e.at)} · {timeOnly(e.at)}
            </span>
          </div>
          <div className="mt-0.5 flex gap-3 font-mono text-[11px] text-muted-foreground">
            {PROVIDERS.map((p) =>
              e.quota[p] != null ? (
                <span key={p} className="flex items-center gap-1">
                  <span
                    className="size-1.5 rounded-full"
                    style={{ backgroundColor: PROVIDER_COLORS[p] }}
                    aria-hidden="true"
                  />
                  {p} {pct(e.quota[p])}
                </span>
              ) : null,
            )}
          </div>
        </div>
      ))}
    </div>
  )
}

function RecentSnapshots({ email }: { email: string }) {
  const { data, isPending } = useSnapshots(email, 10)
  if (isPending) return <Skeleton className="h-24 w-full" />
  const snapshots = data?.snapshots ?? []
  if (snapshots.length === 0) {
    return <p className="text-xs text-muted-foreground">No snapshots recorded.</p>
  }
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Captured</TableHead>
          <TableHead className="text-right">Prompt</TableHead>
          <TableHead className="text-right">Flow</TableHead>
          <TableHead className="text-right">Models</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {snapshots.map((s) => (
          <TableRow key={s.captured_at}>
            <TableCell className="tnum font-mono text-xs">
              {shortDateTime(s.captured_at)}
            </TableCell>
            <TableCell className="tnum text-right font-mono text-xs">
              {s.prompt_credits_available}/{s.prompt_credits_monthly}
            </TableCell>
            <TableCell className="tnum text-right font-mono text-xs">
              {s.flow_credits_available}/{s.flow_credits_monthly}
            </TableCell>
            <TableCell className="tnum text-right font-mono text-xs">{s.models.length}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

/** Right slide-over with the full picture for one account. */
export function AccountSheet({
  account,
  masked,
  onClose,
}: {
  account: apiclient.Account | null
  masked: boolean
  onClose: () => void
}) {
  const snapshot = account?.latest_snapshot
  return (
    <Sheet open={account !== null} onOpenChange={(open) => !open && onClose()}>
      <SheetContent>
        {account && (
          <>
            <SheetHeader>
              <SheetTitle>{masked ? maskEmail(account.email) : account.email}</SheetTitle>
              <SheetDescription>
                {account.plan_name || 'Unknown plan'} · first seen{' '}
                {shortDateTime(account.first_seen)}
                {snapshot && (
                  <>
                    {' · '}
                    <span title={shortDateTime(snapshot.captured_at)}>
                      polled {ago(snapshot.staleness_seconds)} ago
                    </span>
                  </>
                )}
              </SheetDescription>
            </SheetHeader>
            <div className="flex-1 space-y-8 overflow-y-auto p-5">
              <section>
                <SectionTitle>Model quotas</SectionTitle>
                <ModelQuotas email={account.email} />
              </section>
              <section>
                <SectionTitle>Login history</SectionTitle>
                <Timeline email={account.email} />
              </section>
              <section>
                <SectionTitle>Last 10 snapshots</SectionTitle>
                <RecentSnapshots email={account.email} />
              </section>
            </div>
          </>
        )}
      </SheetContent>
    </Sheet>
  )
}
