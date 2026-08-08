/**
 * Provider grouping, mirroring the monitor's classifyProvider so client-side
 * grouping (account cards) matches server-side aggregates (analytics).
 */

export const PROVIDERS = ['Gemini', 'Anthropic', 'OpenAI'] as const
export type Provider = (typeof PROVIDERS)[number]

export const PROVIDER_COLORS: Record<Provider, string> = {
  Gemini: 'var(--chart-gemini)',
  Anthropic: 'var(--chart-anthropic)',
  OpenAI: 'var(--chart-openai)',
}

export function classifyProvider(label: string): Provider | null {
  const l = label.toLowerCase()
  if (l.includes('gemini')) return 'Gemini'
  if (
    l.includes('claude') ||
    l.includes('anthropic') ||
    l.includes('sonnet') ||
    l.includes('opus') ||
    l.includes('haiku')
  ) {
    return 'Anthropic'
  }
  if (l.includes('gpt') || l.includes('openai') || l.includes('codex')) return 'OpenAI'
  return null
}

/** Groups items by provider, preserving PROVIDERS order and dropping unknowns. */
export function groupByProvider<T>(
  items: T[],
  label: (item: T) => string,
): { provider: Provider; items: T[] }[] {
  const groups = new Map<Provider, T[]>()
  for (const item of items) {
    const provider = classifyProvider(label(item))
    if (!provider) continue
    const list = groups.get(provider) ?? []
    list.push(item)
    groups.set(provider, list)
  }
  return PROVIDERS.filter((p) => groups.has(p)).map((provider) => ({
    provider,
    items: groups.get(provider)!,
  }))
}
