import { cn } from '@/lib/utils'

/**
 * Thin horizontal bar for a remaining-quota fraction. Color shifts as quota
 * depletes: normal, amber under 25%, red under 10% or exhausted.
 */
export function QuotaBar({
  fraction,
  className,
}: {
  fraction: number | null | undefined
  className?: string
}) {
  const value = fraction == null ? 0 : Math.max(0, Math.min(1, fraction))
  const tone =
    fraction == null
      ? 'bg-muted-foreground/30'
      : value < 0.1
        ? 'bg-destructive'
        : value < 0.25
          ? 'bg-warning'
          : 'bg-primary'

  return (
    <div className={cn('h-1 w-full overflow-hidden rounded-full bg-muted', className)}>
      <div
        className={cn('h-full rounded-full transition-all', tone)}
        style={{ width: `${value * 100}%` }}
      />
    </div>
  )
}
