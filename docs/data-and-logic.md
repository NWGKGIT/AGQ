# Data Model and Core Logic

This document describes AGQ's snapshot model, quota reset cycles, assumed-refill logic, and session inference.

## Snapshot Model

A **snapshot** is a single poll result from one language server process, captured at a specific instant in time. It contains:

```
QuotaSnapshot {
  Email                 string         // Account email from GetUserStatus
  PlanName              string         // User tier name (e.g., "Free", "Pro")
  CapturedAt            time.Time      // UTC timestamp when polled
  PromptCreditsAvailable int64         // Prompt credits balance
  PromptCreditsMonthly   int64         // Prompt credits monthly quota
  FlowCreditsAvailable   int64         // Flow credits balance
  FlowCreditsMonthly     int64         // Flow credits monthly quota
  Models                []ModelQuota   // Per-model quota rows
  RawJSON               string         // Original response (preserved but not used by monitor)
}
```

Each **model quota** describes one AI model's remaining capacity:

```
ModelQuota {
  Label               string         // Human-readable model name (e.g., "Claude 3.5 Sonnet")
  ModelID             string         // Model identifier from provider (e.g., "claude-3-5-sonnet-20241022")
  RemainingFraction   *float64       // Fraction remaining [0.0, 1.0], nil if unknown
  RemainingPct        *float64       // Percentage [0%, 100%], computed from fraction
  IsExhausted         bool           // True if RemainingFraction == 0
  ResetTime           *time.Time     // When this model's quota resets (UTC), nil if no reset scheduled
  PoolResetTime       *time.Time     // Same as ResetTime (for compatibility)
  TimeUntilResetMs    *int64         // Milliseconds until ResetTime, computed at capture time
}
```

## Quota Reset Cycles

A **reset cycle** is the period between two consecutive quota resets for a given model in a given account. Within a cycle:

- The model starts at some `StartingFraction` (captured at the first snapshot in the cycle)
- Over time, the fraction decreases as the model is used
- At the reset time, the fraction resets to 1.0 (100%)

The `ResetTime` field in each `ModelQuota` row indicates when the current cycle ends. All rows captured before that time are part of the same cycle; rows after the reset time are part of the next cycle.

## Assumed-Refill Logic

**Problem:** If AGQ is offline or stopped during a quota reset, when it restarts and queries the database, it may display stale "depleted" numbers for models whose quota already reset server-side.

**Solution:** At serve time (not write time), AGQ checks each model's `ResetTime` against the current time. If the reset time has passed, the model is treated as refilled:

- Fraction is rewritten to 1.0 (100%)
- Flag `assumed_refilled: true` is set in the response
- Consumption figures are cleared (no data from past cycles)

This ensures that stale data never appears "stuck" on the dashboard — a reset cycle always appears complete.

**Implementation:** The `refill.go` handler applies this logic to served model snapshots, account current quotas, and aggregate responses. The database is never rewritten; only the JSON response is transformed at serve time.

## Account Deduplication

When multiple language server processes are polled in one cycle, they may report different emails (e.g., old process with old account, new process with new account after switch). The poller deduplicates by email:

```
results := make(map[string]*QuotaSnapshot)
for each process:
  snap := poll(process)
  results[snap.Email] = snap  // Last write wins for duplicate emails
```

This means if two processes return the same email, the second snapshot overwrites the first. In practice, this occurs rarely (processes are distinct per account), but can happen if:
- Antigravity takes time to fully switch accounts
- The same email is used across multiple workspace processes

**Limitation:** AGQ cannot track which process produced which email. If multiple distinct processes legitimately report the same account, only one is persisted per poll cycle.

## Session Inference

AGQ infers login/logout sessions by looking for gaps in the snapshot timeline. The **session gap threshold** is 10 minutes:

- If consecutive snapshots are ≤ 10 minutes apart, they belong to the same session
- If a gap is > 10 minutes, a logout occurred on the earlier snapshot and a login occurred on the later one

This heuristic sits well above the poller's 5-minute failure backoff, so a run of failed polls is not mistaken for a logout.

**Timeline events** are returned as pairs:

```
[
  { type: "login",  at: "2026-07-15T09:00:00Z", quota: { Gemini: 0.75, Anthropic: 0.50, ... } },
  { type: "logout", at: "2026-07-15T17:30:00Z", quota: { Gemini: 0.60, Anthropic: 0.45, ... } },
  { type: "login",  at: "2026-07-16T08:45:00Z", quota: { Gemini: 1.00, Anthropic: 1.00, ... } },
  ...
]
```

Each event captures the quota state (average per provider) at that boundary.

## Polling Cadence

The poller runs on two intervals:

| State | Interval | Trigger |
|-------|----------|---------|
| Normal | 60 seconds | Active process(es) detected |
| Backoff | 5 minutes | 5+ consecutive all-failure cycles |
| Idle | Never | No processes detected |

A **failure cycle** occurs when every detected process fails to poll (all return errors). After 5 such cycles, the interval backs off to 5 minutes. The first successful poll restores 60-second cadence.

This balancing act:
- Keeps data fresh when the language server is healthy
- Avoids hammering a broken endpoint
- Gracefully degrades when Antigravity is offline

## Data Flow

```
Detector (15s scan)
  ↓
[ProcessInfo list]
  ↓
Poller (60s/5m poll)
  ↓
[QuotaSnapshot per email]
  ↓
Store (SQLite write, transaction per poll)
  ↓
[Accounts, Snapshots, ModelQuotas tables]
  ↓
API Server (read-only, assume-refill at serve time)
  ↓
JSON responses (frontend / external tools)
```

## Database Schema

Three tables:

**accounts**
- `id` — primary key
- `email` — unique account email
- `plan_name` — user tier name
- `first_seen`, `last_seen` — timestamps

**snapshots**
- `id` — primary key
- `account_id` — foreign key to accounts
- `captured_at` — poll timestamp (indexed with account_id)
- `prompt_credits_available`, `prompt_credits_monthly` — credit balances
- `flow_credits_available`, `flow_credits_monthly` — credit balances
- `raw_json` — original response (not used)

**model_quotas**
- `id` — primary key
- `snapshot_id` — foreign key to snapshots
- `label`, `model_id` — model identifier
- `remaining_fraction`, `remaining_pct` — quota level
- `is_exhausted` — boolean
- `reset_time`, `pool_reset_time` — reset timestamps
- `time_until_reset_ms` — computed at insert time

All writes are transactional; all reads use indices on (account_id, captured_at) and (snapshot_id) for fast access.
