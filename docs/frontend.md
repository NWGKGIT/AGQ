# Frontend Architecture & Components

AGQ's frontend is a React SPA (single-page app) bundled with the Wails desktop shell. It communicates with the embedded Go monitor through typed Wails bindings.

## Design System

**Theme:** Dark mode by default, light mode and system preference available. CSS variables control all colors and spacing.

**Chart colors** (CSS variables):
- `--chart-gemini` - Blue-purple (oklch 0.623 0.19 259.8 dark, 0.678 0.16 light)
- `--chart-anthropic` - Orange (oklch 0.705 0.15 40.24 dark, 0.745 0.14 light)
- `--chart-openai` - Cyan (oklch 0.696 0.14 165.46 dark, 0.741 0.13 light)

**Component library:** shadcn/ui (Radix UI primitives + Tailwind CSS). Custom UI lives in `src/components/ui/`.

## Page Structure

### Overview Page (`src/pages/overview.tsx`)

Landing page showing:
- **Status Header** - daemon state, active account, last poll time, next reset
- **Provider Strip** - per-provider aggregate: average remaining quota, model count, account count
- **Account Cards** - one card per account with latest model breakdown and quotas

### Analytics Page (`src/pages/analytics.tsx`)

Data insights:
- **Stat Cards** - headline metrics (next reset, most depleted model, healthiest account)
- **Usage Chart** - per-day remaining quota trend (7d/30d, avg/min aggregation)
- **Breakdown Table** - consumption per account/model, sortable by consumed/model/provider

### Settings Page (`src/pages/settings.tsx`)

Configuration:
- Email masking toggle (masks screenshots for privacy)
- API exposure toggle (enables TCP listener)
- Theme selector

## Component Map

```
src/
├── components/
│   ├── ui/              # shadcn primitives (badge, button, card, etc.)
│   ├── overview/
│   │   ├── status-header.tsx      # State, active email, poll cadence
│   │   ├── provider-strip.tsx     # Aggregate per provider
│   │   ├── account-sheet.tsx      # Account detail modal
│   │   └── account-card.tsx       # Account card with model list
│   ├── analytics/
│   │   ├── usage-chart.tsx        # Area chart, range/agg toggles
│   │   ├── stat-cards.tsx         # Headline metrics
│   │   └── breakdown-table.tsx    # Model consumption breakdown
│   ├── theme-provider.tsx         # Context + localStorage persistence
│   └── app.tsx                    # Main app shell
├── lib/
│   ├── api.ts                     # React Query hooks (useTimeseries, etc.)
│   ├── providers.ts               # Provider classification & grouping
│   ├── format.ts                  # Formatting utilities (ago, until, maskEmail)
│   └── utils.ts                   # Generic utilities (cn, etc.)
├── pages/
│   ├── overview.tsx               # Landing page
│   ├── analytics.tsx              # Analytics dashboard
│   └── settings.tsx               # Settings
└── index.css                      # Global styles + theme variables
```

## Data Flow

**Polling cadence:**
- All live monitor data: 2-second fallback polling, plus immediate Wails update events
- All other queries: 30 seconds (half the backend's 60s poll interval)

**Data fetching** uses React Query (`@tanstack/react-query`):
- Automatic refetch on configured intervals
- Stale-while-revalidate caching
- Automatic retry with exponential backoff
- Request deduplication

**Example:** `useTimeseries(range, agg)` hook:
```typescript
export function useTimeseries(range: '7d' | '30d', agg: 'avg' | 'min') {
  return useQuery({
    queryKey: ['timeseries', range, agg],
    queryFn: () => GetTimeseries(range, agg),
    refetchInterval: REFETCH_MS,  // 2 seconds; events normally update sooner
    retry: false,
  })
}
```

When data changes in the backend, the frontend refetches automatically and updates. If the monitor becomes unreachable, hooks return `isPending: true` and display a skeleton loader.

## Provider Classification

All analytics endpoints and UI components group models by provider using consistent classification:

```typescript
export function classifyProvider(label: string): Provider | null {
  const l = label.toLowerCase()
  if (l.includes('gemini')) return 'Gemini'
  if (l.includes('claude') || l.includes('anthropic') || 
      l.includes('sonnet') || l.includes('opus') || l.includes('haiku'))
    return 'Anthropic'
  if (l.includes('gpt') || l.includes('openai') || l.includes('codex'))
    return 'OpenAI'
  return null
}
```

This mirrors the backend's `classifyProvider()` in `internal/api/provider.go` so client and server grouping match exactly.

## Charts

**Usage Chart** (`src/components/analytics/usage-chart.tsx`):
- Recharts `AreaChart` with three filled areas (one per provider)
- Toggles for range (7d / 30d) and aggregation (avg / min)
- Tooltips on hover, legend at bottom
- Null gaps handled by `connectNulls` prop (interpolates across missing data)

**Data transformation:** Backend returns `days[].providers{Gemini, Anthropic, OpenAI}` with `null` for absent providers. Frontend maps to chart-friendly structure:

```typescript
const chartData = data.days.map(day => ({
  date: formatDate(day.date),
  Gemini: day.providers['Gemini'] != null ? day.providers['Gemini'] * 100 : null,
  Anthropic: day.providers['Anthropic'] != null ? day.providers['Anthropic'] * 100 : null,
  OpenAI: day.providers['OpenAI'] != null ? day.providers['OpenAI'] * 100 : null,
}))
```

## Error Handling

- **Monitor unreachable** - displays offline message, shows last known state if available
- **Query errors** - retry logic built into React Query; if retries exhaust, show error toast
- **Invalid data** - TypeScript types catch schema mismatches at build time

## Performance

- **Code splitting** - each page is lazy-loaded via React Router
- **Image optimization** - no images (UI is text + charts)
- **Bundle size** - Vite build produces ~180 KB gzipped (React, Recharts, shadcn)
- **Time to interactive** - ~500 ms on modern machines (Wails launches with precompiled JS)

## Accessibility

- Semantic HTML (`<button>`, `<nav>`, `<main>`)
- ARIA labels where needed (chart legend)
- Keyboard navigation (Tab/Enter for controls)
- Color contrast meets WCAG AA (chart colors tested)

## Development

**Local dev:**
```bash
make desktop-dev
```

Runs Wails in dev mode with hot reload on both frontend and backend changes.

**TypeScript:** Strict mode enabled; bindings auto-generated from Go.

**Linting:** ESLint + Prettier configured in `desktop/`.
