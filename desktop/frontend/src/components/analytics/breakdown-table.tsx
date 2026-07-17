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
import { useBreakdown } from '@/lib/api'
import { maskEmail, pct, until } from '@/lib/format'
import { PROVIDERS, classifyProvider, type Provider } from '@/lib/providers'
import { cn } from '@/lib/utils'
import type { apiclient } from '../../../wailsjs/go/models'

type SortKey = 'email' | 'provider' | 'label' | 'starting' | 'current' | 'consumed' | 'reset'

type Row = apiclient.BreakdownRow & { provider: Provider | null }

const columns: { key: SortKey; title: string; align?: 'right' }[] = [
  { key: 'email', title: 'Account' },
  { key: 'provider', title: 'Provider' },
  { key: 'label', title: 'Model' },
  { key: 'starting', title: 'Start' },
  { key: 'current', title: 'Current' },
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
      <span className="font-mono text-xs">{pct(consumed)}</span>
    </div>
  )
}

/** Account x model consumption table with client-side sort and provider filter. */
export function BreakdownTable({ masked }: { masked: boolean }) {
  const { data, isPending } = useBreakdown()
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
        <h2 className="text-xs font-medium uppercase tracking-widest text-muted-foreground">
          Per-account breakdown
        </h2>
        <div className="flex gap-1">
          <button
            onClick={() => setProviderFilter(null)}
            className={cn(
              'rounded-md px-2 py-1 text-xs transition-colors',
              providerFilter === null
                ? 'bg-accent text-accent-foreground'
                : 'text-muted-foreground hover:text-foreground',
            )}
          >
            All
          </button>
          {PROVIDERS.map((p) => (
            <button
              key={p}
              onClick={() => setProviderFilter(providerFilter === p ? null : p)}
              className={cn(
                'rounded-md px-2 py-1 text-xs transition-colors',
                providerFilter === p
                  ? 'bg-accent text-accent-foreground'
                  : 'text-muted-foreground hover:text-foreground',
              )}
            >
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
                <TableCell className="text-xs">{row.provider ?? '–'}</TableCell>
                <TableCell className="text-xs" title={row.model_id}>
                  {row.label}
                </TableCell>
                <TableCell className="font-mono text-xs">{pct(row.starting_fraction)}</TableCell>
                <TableCell
                  className={cn(
                    'font-mono text-xs',
                    row.current_fraction != null && row.current_fraction < 0.1 && 'text-destructive',
                  )}
                >
                  {pct(row.current_fraction)}
                </TableCell>
                <TableCell>
                  <ConsumedCell consumed={row.consumed} />
                </TableCell>
                <TableCell className="text-right font-mono text-xs">
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
