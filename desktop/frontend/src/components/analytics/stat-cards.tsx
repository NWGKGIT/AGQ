import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useStats } from '@/lib/api'
import { maskEmail, pct, until } from '@/lib/format'
import { classifyProvider } from '@/lib/providers'

function StatCard({
  label,
  value,
  detail,
  mono = false,
}: {
  label: string
  value: React.ReactNode
  detail?: React.ReactNode
  mono?: boolean
}) {
  return (
    <Card className="flex h-[104px] flex-col justify-center gap-1 p-4">
      <span className="text-xs font-medium uppercase tracking-widest text-muted-foreground">
        {label}
      </span>
      <span className={mono ? 'font-mono text-2xl font-bold tracking-tight' : 'truncate text-base font-medium'}>
        {value}
      </span>
      {detail && <span className="truncate text-xs text-muted-foreground">{detail}</span>}
    </Card>
  )
}

/** The four server-computed headline figures. Null figures render as em-dash. */
export function StatCards({ masked }: { masked: boolean }) {
  const { data, isPending } = useStats()
  const display = (email: string) => (masked ? maskEmail(email) : email)

  if (isPending) {
    return (
      <section className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        {Array.from({ length: 4 }, (_, i) => (
          <Skeleton key={i} className="h-[104px]" />
        ))}
      </section>
    )
  }

  const depleted = data?.most_depleted_model
  const remaining = data?.account_most_remaining
  const reset = data?.next_reset

  return (
    <section className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      <StatCard
        label="Polls this week"
        value={data ? data.total_polls_this_week.toLocaleString() : '—'}
        mono
      />
      <StatCard
        label="Most depleted model"
        value={depleted ? depleted.label : '—'}
        detail={
          depleted
            ? `${classifyProvider(depleted.label) ?? 'Unknown'} · ${pct(
                depleted.remaining_fraction,
              )} left · ${display(depleted.email)}`
            : 'no data'
        }
      />
      <StatCard
        label="Most remaining account"
        value={remaining ? display(remaining.email) : '—'}
        detail={remaining ? `${pct(remaining.remaining_fraction)} average remaining` : 'no data'}
      />
      <StatCard
        label="Next reset"
        value={reset ? until(reset.reset_time) : '—'}
        detail={reset ? `${reset.label} · ${display(reset.email)}` : 'no data'}
        mono
      />
    </section>
  )
}
