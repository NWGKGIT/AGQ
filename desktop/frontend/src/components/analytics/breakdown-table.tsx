import { useMemo, useState } from 'react'
import { ArrowDown, ArrowUp, ArrowUpDown } from 'lucide-react'

import { Skeleton } from '@/components/ui/skeleton'
import { Card } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useBreakdown, useStats } from '@/lib/api'
import { maskEmail, pct, until } from '@/lib/format'
import { PROVIDERS, PROVIDER_COLORS, classifyProvider, type Provider } from '@/lib/providers'
import { cn } from '@/lib/utils'
import type { apiclient } from '../../../wailsjs/go/models'

type SortKey = 'email' | 'provider' | 'label' | 'starting' | 'current' | 'consumed' | 'reset'

type Row = apiclient.BreakdownRow & { provider: Provider | null }

const columns: { key: SortKey; title: string; align?: 'right' }[] = [
  { key: 'email', title: 'Account' },
  { key: 'label', title: 'Model' },
  { key: 'current', title: 'Remaining' },
  { key: 'starting', title: 'Start' },
  { key: 'consumed', title: 'Consumed' },
  { key: 'reset', title: 'Resets', align: 'right' },
]

// Sort accessors; undefined-ish values always sort last regardless of order.
function accessor(row: Row, key: SortKey): string | number | null {
  switch (key) {
    case 'email':
      return row.email
    case 'provider':
      return row.provider
    case 'label':
      return row.label
    case 'starting':
      return row.starting_fraction ?? null
    case 'current':
      return row.current_fraction ?? null
    case 'consumed':
      return row.consumed ?? null
    case 'reset':
      return row.reset_time ?? null
  }
}

/** Remaining quota as a provider-colored bar with the percentage beside it. */
function RemainingCell({ row }: { row: Row }) {
  const fraction = row.current_fraction
  if (fraction == null) return <span className="text-muted-foreground">–</span>
  const clamped = Math.max(0, Math.min(1, fraction))
  const low = clamped < 0.1
  const color = low
    ? 'var(--destructive)'
    : row.provider
      ? PROVIDER_COLORS[row.provider]
      : 'var(--muted-foreground)'
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 w-20 overflow-hidden rounded-full bg-muted">
        <div
          className="h-full rounded-full transition-all duration-300"
          style={{ width: `${clamped * 100}%`, backgroundColor: color }}
        />
      </div>
      <span className={cn('tnum font-mono text-xs', low && 'font-semibold text-destructive')}>
        {pct(fraction)}
      </span>
      {row.assumed_refilled && (
        <span
          className="rounded-full bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground"
          title="The reset time passed with no fresh poll; quota is assumed refilled."
        >
          assumed
        </span>
      )}
    </div>
  )
}

function ConsumedCell({ consumed }: { consumed: number | null | undefined }) {
  if (consumed == null) return <span className="text-muted-foreground">–</span>
  const clamped = Math.max(0, Math.min(1, consumed))
  return (
    <div className="flex items-center gap-2">
      <div className="h-1 w-16 overflow-hidden rounded-full bg-muted">
        <div
          className={cn('h-full', clamped > 0.75 ? 'bg-destructive' : 'bg-foreground/60')}
          style={{ width: `${clamped * 100}%` }}
        />
      </div>
      <span className="tnum font-mono text-xs">{pct(consumed)}</span>
    </div>
  )
}

/** Account x model consumption table with client-side sort and provider filter. */
export function BreakdownTable({ masked }: { masked: boolean }) {
  const { data, isPending } = useBreakdown()
  const { data: stats } = useStats()
  const [sortKey, setSortKey] = useState<SortKey>('consumed')
  const [descending, setDescending] = useState(true)
  const [providerFilter, setProviderFilter] = useState<Provider | null>(null)

  const rows = useMemo<Row[]>(() => {
    let out = (data?.rows ?? []).map((r) => ({ ...r, provider: classifyProvider(r.label) }))
    if (providerFilter) out = out.filter((r) => r.provider === providerFilter)
    out.sort((a, b) => {
      const av = accessor(a, sortKey)
      const bv = accessor(b, sortKey)
      if (av == null && bv == null) return 0
      if (av == null) return 1
      if (bv == null) return -1
      const cmp = typeof av === 'number' ? av - (bv as number) : String(av).localeCompare(String(bv))
      return descending ? -cmp : cmp
    })
    return out
  }, [data, sortKey, descending, providerFilter])

  const toggleSort = (key: SortKey) => {
    if (key === sortKey) {
      setDescending((d) => !d)
    } else {
      setSortKey(key)
      setDescending(key !== 'email' && key !== 'provider' && key !== 'label' && key !== 'reset')
    }
  }

  if (isPending) return <Skeleton className="h-64" />

  return (
    <Card className="overflow-hidden">
      <div className="flex items-center justify-between border-b p-4">
        <div className="flex items-baseline gap-3">
          <h2 className="text-xs font-medium uppercase tracking-widest text-muted-foreground">
            Per-model breakdown
          </h2>
          {stats && (
            <span className="tnum font-mono text-[11px] text-muted-foreground/70">
              {stats.total_polls_this_week.toLocaleString()} polls this week
            </span>
          )}
        </div>
        <div className="flex gap-1">
          <button
            onClick={() => setProviderFilter(null)}
            className={cn(
              'rounded-md px-2 py-1 text-xs transition-colors duration-150',
              providerFilter === null
                ? 'bg-accent text-accent-foreground'
                : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground',
            )}
          >
            All
          </button>
          {PROVIDERS.map((p) => (
            <button
              key={p}
              onClick={() => setProviderFilter(providerFilter === p ? null : p)}
              className={cn(
                'flex items-center gap-1.5 rounded-md px-2 py-1 text-xs transition-colors duration-150',
                providerFilter === p
                  ? 'bg-accent text-accent-foreground'
                  : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground',
              )}
            >
              <span
                className="size-1.5 rounded-full"
                style={{ backgroundColor: PROVIDER_COLORS[p] }}
                aria-hidden="true"
              />
              {p}
            </button>
          ))}
        </div>
      </div>
      {rows.length === 0 ? (
        <p className="p-8 text-center text-sm text-muted-foreground">
          No quota rows yet{providerFilter ? ` for ${providerFilter}` : ''}.
        </p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              {columns.map(({ key, title, align }) => (
                <TableHead key={key} className={align === 'right' ? 'text-right' : ''}>
                  <button
                    onClick={() => toggleSort(key)}
                    className={cn(
                      'inline-flex items-center gap-1 uppercase tracking-wider transition-colors hover:text-foreground',
                      sortKey === key && 'text-foreground',
                    )}
                  >
                    {title}
                    {sortKey === key ? (
                      descending ? (
                        <ArrowDown className="size-3" />
                      ) : (
                        <ArrowUp className="size-3" />
                      )
                    ) : (
                      <ArrowUpDown className="size-3 opacity-40" />
                    )}
                  </button>
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row) => (
              <TableRow key={`${row.email}-${row.model_id}-${row.label}`}>
                <TableCell className="font-mono text-xs text-muted-foreground">
                  {masked ? maskEmail(row.email) : row.email}
                </TableCell>
                <TableCell className="text-xs" title={row.model_id}>
                  <span className="flex items-center gap-1.5">
                    <span
                      className="size-1.5 shrink-0 rounded-full"
                      style={{
                        backgroundColor: row.provider
                          ? PROVIDER_COLORS[row.provider]
                          : 'var(--muted-foreground)',
                      }}
                      aria-hidden="true"
                    />
                    {row.label}
                  </span>
                </TableCell>
                <TableCell>
                  <RemainingCell row={row} />
                </TableCell>
                <TableCell className="tnum font-mono text-xs">
                  {pct(row.starting_fraction)}
                </TableCell>
                <TableCell>
                  <ConsumedCell consumed={row.consumed} />
                </TableCell>
                <TableCell className="tnum text-right font-mono text-xs">
                  {row.reset_time ? until(row.reset_time) : '–'}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </Card>
  )
}
