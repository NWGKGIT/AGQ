import { useMemo } from 'react'

import { QuotaBar } from '@/components/quota-bar'
import { Skeleton } from '@/components/ui/skeleton'
import { useModelsLatest } from '@/lib/api'
import { pct } from '@/lib/format'
import { PROVIDERS, PROVIDER_COLORS, classifyProvider, type Provider } from '@/lib/providers'
import type { apiclient } from '../../../wailsjs/go/models'

type ProviderAggregate = {
  provider: Provider
  fraction: number | null
  models: number
  accounts: number
}

function aggregate(models: apiclient.ModelAggregate[]): ProviderAggregate[] {
  const byProvider = new Map<Provider, apiclient.ModelAggregate[]>()
  for (const m of models) {
    const provider = classifyProvider(m.label)
    if (!provider) continue
    const list = byProvider.get(provider) ?? []
    list.push(m)
    byProvider.set(provider, list)
  }

  return PROVIDERS.map((provider) => {
    const rows = byProvider.get(provider) ?? []
    const fractions = rows
      .map((r) => r.remaining_fraction)
      .filter((f): f is number => f != null)
    return {
      provider,
      fraction:
        fractions.length > 0 ? fractions.reduce((a, b) => a + b, 0) / fractions.length : null,
      models: new Set(rows.map((r) => r.model_id || r.label)).size,
      accounts: new Set(rows.map((r) => r.email)).size,
    }
  })
}

/** Per-provider aggregate blocks across all accounts' newest snapshots. */
export function ProviderStrip() {
  const { data, isPending } = useModelsLatest()
  const aggregates = useMemo(() => aggregate(data?.models ?? []), [data])

  return (
    <section className="grid grid-cols-1 divide-y rounded-lg border bg-card sm:grid-cols-3 sm:divide-x sm:divide-y-0">
      {isPending
        ? PROVIDERS.map((p) => (
            <div key={p} className="space-y-2 p-5">
              <Skeleton className="h-3 w-16" />
              <Skeleton className="h-8 w-20" />
              <Skeleton className="h-1 w-full" />
            </div>
          ))
        : aggregates.map(({ provider, fraction, models, accounts }) => (
            <div
              key={provider}
              className="interactive-surface flex flex-col gap-1.5 border-transparent p-5 first:rounded-l-lg last:rounded-r-lg"
            >
              <span className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-widest text-muted-foreground">
                <span
                  className="size-1.5 rounded-full"
                  style={{ backgroundColor: PROVIDER_COLORS[provider] }}
                  aria-hidden="true"
                />
                {provider}
              </span>
              <span className="tnum font-mono text-3xl font-bold tracking-tight">
                {pct(fraction)}
              </span>
              <QuotaBar fraction={fraction} className="h-[3px]" />
              <span className="text-xs text-muted-foreground">
                {models} {models === 1 ? 'model' : 'models'} · {accounts}{' '}
                {accounts === 1 ? 'account' : 'accounts'}
              </span>
            </div>
          ))}
    </section>
  )
}
