import { Sparkline } from '@/components/sparkline'
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
import { useSnapshots, useSparklines, useTimeline } from '@/lib/api'
import { maskEmail, pct, shortDateTime, timeOnly, until } from '@/lib/format'
import { PROVIDERS, groupByProvider } from '@/lib/providers'
import { cn } from '@/lib/utils'
import type { apiclient } from '../../../wailsjs/go/models'

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <h3 className="mb-3 text-xs font-medium uppercase tracking-widest text-muted-foreground">
      {children}
    </h3>
  )
}

function PoolCards({ models }: { models: apiclient.ModelQuota[] }) {
  const groups = groupByProvider(models, (m) => m.label)
  if (groups.length === 0) return null
  return (
    <section className="grid grid-cols-2 gap-3">
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
          <div key={provider} className="rounded-lg border bg-background p-3">
            <div className="mb-1 flex items-center justify-between">
              <span className="text-xs uppercase tracking-wide text-muted-foreground">
                {provider} pool
              </span>
              <span className="font-mono text-sm font-bold">{pct(avg)}</span>
            </div>
            <div className="font-mono text-[11px] text-muted-foreground">
              {reset ? `resets in ${until(reset)}` : 'no reset scheduled'}
            </div>
          </div>
        )
      })}
    </section>
  )
}

function Sparklines({ email }: { email: string }) {
  const { data, isPending } = useSparklines(email)
  if (isPending) return <Skeleton className="h-24 w-full" />
  const models = data?.models ?? []
  if (models.length === 0) {
    return <p className="text-xs text-muted-foreground">No history in the last 7 days.</p>
  }
  return (
    <div className="flex flex-col">
      {models.map((m) => {
        const values = m.points.map((p) => p.remaining_fraction ?? null)
        const latest = [...values].reverse().find((v) => v != null) ?? null
        return (
          <div
            key={m.model_id + m.label}
            className="flex items-center justify-between gap-3 border-b py-2 last:border-0"
          >
            <span className="w-1/3 truncate text-sm" title={m.label}>
              {m.label}
            </span>
            <Sparkline values={values} />
            <span className="w-12 text-right font-mono text-sm">{pct(latest)}</span>
          </div>
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
    <div className="relative ml-1 flex flex-col gap-4 border-l pl-4">
      {events.map((e, i) => (
        <div key={`${e.at}-${i}`} className="relative">
          <span
            className={cn(
              'absolute -left-[21.5px] top-1 size-2 rounded-full border-2 border-background',
              e.type === 'login' ? 'bg-success' : 'bg-muted-foreground',
            )}
          />
          <div className="flex items-baseline justify-between gap-2">
            <span className="text-sm capitalize">
              {e.type === 'login' ? 'Logged in' : 'Logged out'}
            </span>
            <span className="font-mono text-[11px] text-muted-foreground">
              {shortDateTime(e.at)} · {timeOnly(e.at)}
            </span>
          </div>
          <div className="mt-0.5 flex gap-3 font-mono text-[11px] text-muted-foreground">
            {PROVIDERS.map((p) =>
              e.quota[p] != null ? (
                <span key={p}>
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
            <TableCell className="font-mono text-xs">{shortDateTime(s.captured_at)}</TableCell>
            <TableCell className="text-right font-mono text-xs">
              {s.prompt_credits_available}/{s.prompt_credits_monthly}
            </TableCell>
            <TableCell className="text-right font-mono text-xs">
              {s.flow_credits_available}/{s.flow_credits_monthly}
            </TableCell>
            <TableCell className="text-right font-mono text-xs">{s.models.length}</TableCell>
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
              </SheetDescription>
            </SheetHeader>
            <div className="flex-1 space-y-8 overflow-y-auto p-5">
              <PoolCards models={account.latest_snapshot?.models ?? []} />
              <section>
                <SectionTitle>Usage history — 7 days</SectionTitle>
                <Sparklines email={account.email} />
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
