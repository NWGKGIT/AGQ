import { describe, expect, it } from 'vitest'

import { deriveHealth, healthStatusForFraction } from './health'
import type { apiclient } from '../../wailsjs/go/models'

function quota(
  fraction: number | null,
  exhausted = false,
): apiclient.ModelQuota {
  return {
    label: 'Model',
    model_id: 'model',
    remaining_fraction: fraction ?? undefined,
    remaining_pct: fraction == null ? undefined : fraction * 100,
    is_exhausted: exhausted,
    reset_time: undefined,
    pool_reset_time: undefined,
    time_until_reset_ms: undefined,
  }
}

describe('healthStatusForFraction', () => {
  it.each([
    [0, 'low'],
    [0.1999, 'low'],
    [0.2, 'warning'],
    [0.5, 'warning'],
    [0.5001, 'good'],
    [1, 'good'],
    [null, 'unknown'],
    [undefined, 'unknown'],
    [Number.NaN, 'unknown'],
  ] as const)('classifies %s as %s', (fraction, status) => {
    expect(healthStatusForFraction(fraction)).toBe(status)
  })
})

describe('deriveHealth', () => {
  it('uses the worst known model', () => {
    expect(deriveHealth([quota(0.8), quota(0.35), quota(0.1)])).toEqual({
      status: 'low',
      lowestRemainingFraction: 0.1,
    })
  })

  it('treats an explicit exhausted flag as low', () => {
    expect(deriveHealth([quota(null, true), quota(0.9)])).toEqual({
      status: 'low',
      lowestRemainingFraction: 0,
    })
  })

  it('returns unknown when no model has quota data', () => {
    expect(deriveHealth([quota(null), quota(Number.NaN)])).toEqual({
      status: 'unknown',
      lowestRemainingFraction: null,
    })
  })
})
