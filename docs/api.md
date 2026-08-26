# HTTP API Reference

AGQ exposes a read-only JSON API on `localhost:7432` (configurable via `AGQ_PORT` env var). The desktop app embeds this API in-process; the headless daemon exposes it over TCP.

All responses use 2-space JSON indentation. Dates are RFC3339 UTC. Fractions are [0.0, 1.0]; percentages are [0, 100].

## Health & Status

### GET /api/health

Health and uptime.

**Response:**

```json
{
  "status": "ok",
  "uptime": "2h15m30s"
}
```

### GET /api/status

Daemon state (whether AGQ is monitoring Antigravity) is separate from
confirmed Antigravity login state (`is_live`). An ACTIVE monitor can retain a
last-known account while Antigravity has not yet produced a fresh confirmed
login.

**Response:**

```json
{
  "state": "ACTIVE",
  "emails": ["user@example.com"],
  "last_poll_at": "2026-07-15T14:30:00Z",
  "next_poll_at": "2026-07-15T14:31:00Z",
  "uptime": "5h30m",
  "started_at": "2026-07-15T09:00:00Z"
}
```

## Accounts

### GET /api/accounts

List all known accounts with their latest snapshots.

**Response:**

```json
{
  "accounts": [
    {
      "id": 1,
      "email": "user@example.com",
      "plan_name": "Pro",
      "first_seen": "2026-07-01T08:00:00Z",
      "last_seen": "2026-07-15T14:30:00Z",
      "latest_snapshot": {
        "captured_at": "2026-07-15T14:30:00Z",
        "prompt_credits_available": 50000,
        "prompt_credits_monthly": 100000,
        "flow_credits_available": 1000,
        "flow_credits_monthly": 2000,
        "models": [
          {
            "label": "Claude 3.5 Sonnet",
            "model_id": "claude-3-5-sonnet-20241022",
            "remaining_fraction": 0.75,
            "remaining_pct": 75.0,
            "is_exhausted": false,
            "reset_time": "2026-08-01T00:00:00Z"
          }
        ]
      }
    }
  ]
}
```

### GET /api/account/current

Current active account (or last known account if idle).

**Response:**

```json
{
  "state": "ACTIVE",
  "is_live": true,
  "email": "user@example.com",
  "account": {
    "id": 1,
    "email": "user@example.com",
    "plan_name": "Pro",
    "first_seen": "2026-07-01T08:00:00Z",
    "last_seen": "2026-07-15T14:30:00Z",
    "latest_snapshot": null
  },
  "accounts": [
    {
      "id": 1,
      "email": "user@example.com",
      "plan_name": "Pro",
      "first_seen": "2026-07-01T08:00:00Z",
      "last_seen": "2026-07-15T14:30:00Z",
      "latest_snapshot": null
    }
  ],
  "last_poll_at": "2026-07-15T14:30:00Z",
  "next_poll_at": "2026-07-15T14:31:00Z",
  "as_of": "2026-07-15T14:30:00Z",
  "last_account": null
}
```

### GET /api/accounts/:email/latest

Latest snapshot for a specific account, with assumed-refill applied.

**Response:**

```json
{
  "email": "user@example.com",
  "plan_name": "Pro",
  "captured_at": "2026-07-15T14:30:00Z",
  "prompt_credits_available": 50000,
  "prompt_credits_monthly": 100000,
  "flow_credits_available": 1000,
  "flow_credits_monthly": 2000,
  "models": [
    {
      "label": "Claude 3.5 Sonnet",
      "model_id": "claude-3-5-sonnet-20241022",
      "remaining_fraction": 0.75,
      "remaining_pct": 75.0,
      "is_exhausted": false,
      "reset_time": "2026-08-01T00:00:00Z"
    }
  ]
}
```

### GET /api/accounts/:email/snapshots?limit=50&before=

Paginated snapshot history for an account, newest-first.

**Query params:**

- `limit` - result count, clamped to [1, 200], default 50
- `before` - RFC3339 timestamp; return snapshots captured before this time, default now

**Response:**

```json
{
  "email": "user@example.com",
  "snapshots": [
    {
      "captured_at": "2026-07-15T14:30:00Z",
      "prompt_credits_available": 50000,
      "models": [...]
    }
  ]
}
```

## Models

### GET /api/models/latest

Latest model quotas across all accounts, deduplicated by model ID.

**Response:**

```json
{
  "models": [
    {
      "label": "Claude 3.5 Sonnet",
      "model_id": "claude-3-5-sonnet-20241022",
      "email": "user@example.com",
      "remaining_fraction": 0.75,
      "remaining_pct": 75.0,
      "is_exhausted": false,
      "reset_time": "2026-08-01T00:00:00Z",
      "captured_at": "2026-07-15T14:30:00Z",
      "staleness_seconds": 120
    }
  ]
}
```

### GET /api/accounts/:email/models/current

Current model quotas for one account, fallback to newest non-null per model within 14 days.

**Response:** Same as `/api/models/latest` but filtered to one email.

### GET /api/accounts/:email/sparklines

Per-model time series (7-day lookback) for dashboard inline charts.

**Response:**

```json
{
  "email": "user@example.com",
  "models": [
    {
      "label": "Claude 3.5 Sonnet",
      "model_id": "claude-3-5-sonnet-20241022",
      "points": [
        {
          "captured_at": "2026-07-08T10:00:00Z",
          "remaining_fraction": 0.5
        },
        {
          "captured_at": "2026-07-08T11:00:00Z",
          "remaining_fraction": null
        },
        {
          "captured_at": "2026-07-08T12:00:00Z",
          "remaining_fraction": 0.48
        }
      ]
    }
  ]
}
```

Null fractions are included so the frontend can render gaps in polling.

## Analytics

### GET /api/analytics/timeseries?range=7d&agg=avg

Per-day remaining quota trend by provider.

**Query params:**

- `range` - `7d` or `30d`, default `7d`
- `agg` - `avg` or `min`, default `avg`

**Response:**

```json
{
  "range": "7d",
  "agg": "avg",
  "days": [
    {
      "date": "2026-07-08",
      "providers": {
        "Gemini": 0.75,
        "Anthropic": 0.5,
        "OpenAI": null
      }
    },
    {
      "date": "2026-07-09",
      "providers": {
        "Gemini": 0.7,
        "Anthropic": 0.52,
        "OpenAI": null
      }
    }
  ]
}
```

Each provider is present in every day's map; null means no data for that provider on that day.

### GET /api/analytics/breakdown

Model consumption analysis (starting fraction − current fraction) for the current reset cycle.

**Response:**

```json
{
  "rows": [
    {
      "email": "user@example.com",
      "label": "Claude 3.5 Sonnet",
      "model_id": "claude-3-5-sonnet-20241022",
      "current_fraction": 0.75,
      "starting_fraction": 1.0,
      "consumed": 0.25,
      "reset_time": "2026-08-01T00:00:00Z",
      "assumed_refilled": false
    },
    {
      "email": "user@example.com",
      "label": "Claude 3 Opus",
      "model_id": "claude-3-opus-20240229",
      "current_fraction": 1.0,
      "starting_fraction": null,
      "consumed": null,
      "reset_time": "2026-07-31T00:00:00Z",
      "assumed_refilled": true
    }
  ]
}
```

`assumed_refilled: true` indicates the cycle has ended; the row shows full capacity (current_fraction: 1.0).

### GET /api/analytics/stats

Headline figures (polls this week, most depleted model, healthiest account, next reset).

**Response:**

```json
{
  "total_polls_this_week": 672,
  "most_depleted_model": {
    "email": "user@example.com",
    "label": "Claude 3.5 Sonnet",
    "remaining_fraction": 0.1
  },
  "account_most_remaining": {
    "email": "user@example.com",
    "remaining_fraction": 0.65
  },
  "next_reset": {
    "email": "user@example.com",
    "label": "Claude 3.5 Sonnet",
    "reset_time": "2026-07-31T00:00:00Z"
  }
}
```

## Timeline

### GET /api/accounts/:email/timeline

Inferred login/logout events with session-boundary quota snapshots (7-day lookback).

**Response:**

```json
{
  "email": "user@example.com",
  "events": [
    {
      "type": "login",
      "at": "2026-07-15T09:00:00Z",
      "quota": {
        "Gemini": 0.75,
        "Anthropic": 0.5,
        "OpenAI": null
      }
    },
    {
      "type": "logout",
      "at": "2026-07-15T17:30:00Z",
      "quota": {
        "Gemini": 0.6,
        "Anthropic": 0.45,
        "OpenAI": null
      }
    }
  ]
}
```

Session gaps > 10 minutes trigger logout/login pairs.

## Error Responses

All errors return JSON with HTTP status codes:

```json
{
  "error": "description of the problem",
  "details": "optional extra info"
}
```

| Status | Meaning                             |
| ------ | ----------------------------------- |
| 400    | Invalid query parameter or request  |
| 404    | Account/model not found             |
| 500    | Internal error (database, encoding) |

## CORS

The API enables CORS for `localhost:*` origins. External tools and scripts can fetch from `http://localhost:7432` (or the configured port) when the desktop app has "Expose API" enabled in Settings.
