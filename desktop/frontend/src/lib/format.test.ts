import { describe, expect, it } from 'vitest'

import { ago, maskEmail, pct, until } from './format'

describe('pct', () => {
  it('formats fractions as whole percents', () => {
    expect(pct(0.42)).toBe('42%')
    expect(pct(1)).toBe('100%')
    expect(pct(0)).toBe('0%')
  })

  it('renders a placeholder for missing values', () => {
    expect(pct(null)).toBe('–')
    expect(pct(undefined)).toBe('–')
  })
})

describe('until', () => {
  const now = new Date('2026-07-01T12:00:00Z')
  const at = (offsetMs: number) => new Date(now.getTime() + offsetMs).toISOString()

  it('never renders 0m: sub-minute futures are <1m', () => {
    expect(until(at(45_000), now)).toBe('<1m')
    expect(until(at(1_000), now)).toBe('<1m')
  })

  it('renders elapsed instants as due', () => {
    expect(until(at(0), now)).toBe('due')
    expect(until(at(-5_000), now)).toBe('due')
  })

  it('renders minutes, hours, and days compactly', () => {
    expect(until(at(12 * 60_000), now)).toBe('12m')
    expect(until(at((4 * 60 + 22) * 60_000), now)).toBe('4h 22m')
    expect(until(at((3 * 24 + 4) * 3_600_000), now)).toBe('3d 4h')
  })

  it('renders a placeholder for missing or invalid input', () => {
    expect(until(null, now)).toBe('–')
    expect(until('not-a-date', now)).toBe('–')
  })
})

describe('ago', () => {
  it('clamps negatives and scales units', () => {
    expect(ago(-5)).toBe('0s')
    expect(ago(12)).toBe('12s')
    expect(ago(4 * 60)).toBe('4m')
    expect(ago(2 * 3600)).toBe('2h')
    expect(ago(3 * 86400)).toBe('3d')
  })
})

describe('maskEmail', () => {
  it('masks the local part but keeps the domain', () => {
    expect(maskEmail('j.doe@gmail.com')).toBe('j***@gmail.com')
  })

  it('leaves malformed addresses untouched', () => {
    expect(maskEmail('not-an-email')).toBe('not-an-email')
  })
})
