import { cn } from '@/lib/utils'
import { healthStatusForFraction } from '@/lib/health'

/**
 * Remaining quota with the same health thresholds used throughout the app.
 */
export function QuotaBar({
  fraction,
  className,
}: {
  fraction: number | null | undefined
  className?: string
}) {
  const known = typeof fraction === 'number' && Number.isFinite(fraction)
  const value = known ? Math.max(0, Math.min(1, fraction)) : 0
  const status = healthStatusForFraction(fraction)
  const tone = {
    low: 'bg-destructive',
    warning: 'bg-warning',
    good: 'bg-success',
    unknown: 'bg-muted-foreground/30',
  }[status]

  return (
    <div
      className={cn('h-1 w-full overflow-hidden rounded-full bg-muted', className)}
      role="progressbar"
      aria-label="Quota remaining"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={known ? Math.round(value * 100) : undefined}
      aria-valuetext={known ? `${Math.round(value * 100)}% remaining, ${status}` : 'Quota unknown'}
    >
      <div
        className={cn('h-full rounded-full transition-all', tone)}
        style={{ width: `${value * 100}%` }}
      />
    </div>
  )
}
