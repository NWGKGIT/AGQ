import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useStats } from '@/lib/api'
import { maskEmail, pct, until } from '@/lib/format'
import { classifyProvider, PROVIDER_COLORS } from '@/lib/providers'

function StatCard({
  label,
  value,
  detail,
  accentColor,
  mono = false,
}: {
  label: string
  value: React.ReactNode
  detail?: React.ReactNode
  accentColor?: string
  mono?: boolean
}) {
  return (
    <Card className="interactive-surface flex h-[104px] flex-col justify-center gap-1 p-4">
      <span className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-widest text-muted-foreground">
        {accentColor && (
          <span
            className="size-1.5 rounded-full"
            style={{ backgroundColor: accentColor }}
            aria-hidden="true"
          />
        )}
        {label}
      </span>
      <span
        className={
          mono
            ? 'tnum font-mono text-2xl font-bold tracking-tight'
            : 'truncate text-base font-medium'
        }
      >
        {value}
      </span>
      {detail && <span className="truncate text-xs text-muted-foreground">{detail}</span>}
    </Card>
  )
}

/** Show the next reset, lowest quota, and healthiest account. */
export function StatCards({ masked }: { masked: boolean }) {
  const { data, isPending } = useStats()
  const display = (email: string) => (masked ? maskEmail(email) : email)

  if (isPending) {
    return (
      <section className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        {Array.from({ length: 3 }, (_, i) => (
          <Skeleton key={i} className="h-[104px]" />
        ))}
      </section>
    )
  }

  const depleted = data?.most_depleted_model
  const remaining = data?.account_most_remaining
  const reset = data?.next_reset
  const depletedProvider = depleted ? classifyProvider(depleted.label) : null

  return (
    <section className="grid grid-cols-1 gap-3 sm:grid-cols-3">
      <StatCard
        label="Next reset"
        value={reset ? until(reset.reset_time) : '-'}
        detail={reset ? `${reset.label} · ${display(reset.email)}` : 'no upcoming reset'}
        mono
      />
      <StatCard
        label="Most depleted model"
        value={depleted ? depleted.label : '-'}
        detail={
          depleted
            ? `${pct(depleted.remaining_fraction)} left · ${display(depleted.email)}`
            : 'no data'
        }
        accentColor={depletedProvider ? PROVIDER_COLORS[depletedProvider] : undefined}
      />
      <StatCard
        label="Healthiest account"
        value={remaining ? display(remaining.email) : '-'}
        detail={remaining ? `${pct(remaining.remaining_fraction)} average remaining` : 'no data'}
      />
    </section>
  )
}
