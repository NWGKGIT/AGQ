/**
 * Minimal SVG sparkline for remaining-fraction series (0..1). Null values
 * break the polyline so data gaps stay visible instead of being interpolated.
 */
export function Sparkline({
  values,
  width = 96,
  height = 24,
}: {
  values: (number | null)[]
  width?: number
  height?: number
}) {
  if (values.length === 0) {
    return <div className="text-xs text-muted-foreground">no data</div>
  }

  const pad = 2
  const step = values.length > 1 ? (width - pad * 2) / (values.length - 1) : 0
  const y = (v: number) => pad + (1 - Math.max(0, Math.min(1, v))) * (height - pad * 2)

  // Split into contiguous non-null segments so gaps render as breaks.
  const segments: string[] = []
  let current: string[] = []
  values.forEach((v, i) => {
    if (v == null) {
      if (current.length > 1) segments.push(current.join(' '))
      current = []
      return
    }
    current.push(`${(pad + i * step).toFixed(1)},${y(v).toFixed(1)}`)
  })
  if (current.length > 1) segments.push(current.join(' '))

  // A single isolated point still deserves a mark.
  const lastValue = values.filter((v): v is number => v != null).at(-1)

  return (
    <svg width={width} height={height} className="shrink-0" role="img" aria-label="usage sparkline">
      {segments.map((points, i) => (
        <polyline
          key={i}
          points={points}
          fill="none"
          stroke="currentColor"
          strokeWidth="1.25"
          className="text-foreground/70"
        />
      ))}
      {segments.length === 0 && lastValue != null && (
        <circle
          cx={width / 2}
          cy={y(lastValue)}
          r="1.5"
          className="fill-foreground/70"
        />
      )}
    </svg>
  )
}
