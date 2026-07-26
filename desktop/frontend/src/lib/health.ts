import type { apiclient } from '../../wailsjs/go/models'

export type HealthStatus = 'low' | 'warning' | 'good' | 'unknown'

export type AccountHealth = {
  status: HealthStatus
  lowestRemainingFraction: number | null
}

export const HEALTH_LABELS: Record<HealthStatus, string> = {
  low: 'Low',
  warning: 'Warning',
  good: 'Good',
  unknown: 'Unknown',
}

export const HEALTH_SORT_ORDER: Record<HealthStatus, number> = {
  low: 0,
  warning: 1,
  unknown: 2,
  good: 3,
}

/** Derive account health from its lowest known model quota. */
export function deriveHealth(models: readonly apiclient.ModelQuota[]): AccountHealth {
	if (models.some((model) => model.is_exhausted)) {
		return { status: 'low', lowestRemainingFraction: 0 }
	}

  const knownFractions = models
    .map((model) => model.remaining_fraction)
    .filter((fraction): fraction is number => typeof fraction === 'number' && Number.isFinite(fraction))

  if (knownFractions.length === 0) {
    return { status: 'unknown', lowestRemainingFraction: null }
  }

  const lowestRemainingFraction = Math.min(...knownFractions)
  const status: HealthStatus =
    lowestRemainingFraction < 0.2
      ? 'low'
      : lowestRemainingFraction <= 0.5
        ? 'warning'
        : 'good'

  return { status, lowestRemainingFraction }
}

export function healthStatusForFraction(fraction: number | null | undefined): HealthStatus {
  if (typeof fraction !== 'number' || !Number.isFinite(fraction)) return 'unknown'
  if (fraction < 0.2) return 'low'
  if (fraction <= 0.5) return 'warning'
  return 'good'
}
