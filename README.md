# AGQ Daemon

AGQ Daemon tracks Antigravity AI quota usage for every authenticated local
language server process and exposes the latest data over a local JSON API.

The daemon is intentionally local-first:

- It discovers Antigravity language servers from `/proc`.
- It probes loopback ports only, using the language server's CSRF token.
- It stores snapshots in `~/.agq/agq.db`.
- It serves the API on `localhost:${AGQ_PORT:-7432}`.

## Build And Run

```sh
make build
make run
```

Run tests with:

```sh
make test
```

If your Go build cache is not writable in a restricted environment, set it to a
temporary directory:

```sh
GOCACHE=/tmp/agq-go-cache go test ./...
```

## Install As A User Service

```sh
make install
make enable
```

Useful service commands:

```sh
make status
make logs
make disable
make uninstall
```

The systemd unit runs `/usr/local/bin/agq-daemon` and appends logs to
`~/.agq/agq.log`.

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `AGQ_PORT` | `7432` | Local HTTP API port. |

## Architecture

The code is organized around narrow packages under `internal/`:

| Package | Responsibility |
| --- | --- |
| `cmd/agq-daemon` | Process setup, logging, signal handling, and wiring. |
| `internal/domain` | Shared domain types. |
| `internal/detector` | `/proc` scanning, command-line parsing, and loopback port discovery. |
| `internal/languageserver` | `GetUserStatus` HTTP client and response parser. |
| `internal/poller` | Poll scheduling, account deduplication, persistence, and daemon status updates. |
| `internal/store` | SQLite schema, migrations, writes, and read queries. |
| `internal/state` | Thread-safe daemon status snapshots. |
| `internal/api` | Local HTTP API handlers and CORS policy. |

Runtime flow:

1. The detector scans `/proc/*/cmdline` every 15 seconds.
2. Matching language server processes are inspected for loopback listening ports.
3. Candidate ports are probed over HTTPS, then HTTP, with `GetUserStatus`.
4. The poller polls active processes every 60 seconds.
5. Successful snapshots are deduplicated by email and persisted to SQLite.
6. The API serves daemon status, accounts, snapshots, and latest model quotas.

After five consecutive all-failure poll cycles, the poller backs off to a
five-minute interval. A later successful poll restores the normal interval.

## API

All endpoints are read-only JSON endpoints. CORS is allowed for
`localhost` and `127.0.0.1` origins.

### `GET /api/health`

Returns daemon liveness and uptime:

```json
{
  "status": "ok",
  "uptime": "1m0s"
}
```

### `GET /api/status`

Returns current daemon state:

```json
{
  "state": "ACTIVE",
  "emails": ["user@example.com"],
  "uptime": "1m0s",
  "started_at": "2026-07-01T12:00:00Z",
  "last_poll_at": "2026-07-01T12:01:00Z",
  "next_poll_at": "2026-07-01T12:02:00Z"
}
```

### `GET /api/accounts`

Returns all known accounts ordered by `last_seen`, including each account's
latest snapshot when available.

### `GET /api/account/current`

Returns the account(s) currently logged in to Antigravity, reflecting live
daemon state joined with the newest persisted snapshot for each. When idle,
`state` is `IDLE` and `accounts` is empty.

```json
{
  "state": "ACTIVE",
  "email": "user@example.com",
  "account": {
    "id": 1,
    "email": "user@example.com",
    "plan_name": "Pro",
    "first_seen": "2026-07-01T10:00:00Z",
    "last_seen": "2026-07-01T11:59:00Z",
    "latest_snapshot": { "staleness_seconds": 15 }
  },
  "accounts": [{ "email": "user@example.com" }],
  "last_poll_at": "2026-07-01T12:01:00Z",
  "as_of": "2026-07-01T12:01:15Z"
}
```

### `GET /api/accounts/{email}/latest`

Returns the latest snapshot for an account, or `404` when no snapshot exists.

### `GET /api/accounts/{email}/snapshots?limit=50&before=<RFC3339>`

Returns snapshot history newest-first. `limit` defaults to 50 and is clamped by
the store to the range `1..200`. `before` is optional.

Invalid `limit` or `before` values return `400`.

### `GET /api/models/latest`

Returns model quota rows from the newest snapshot of each known account.

### `GET /api/analytics/timeseries?range=7d|30d&agg=avg|min`

Returns, for each day in the range, the aggregated remaining fraction per
provider (`Gemini`, `Anthropic`, `OpenAI`) across all accounts. `range` defaults
to `7d`; `agg` defaults to `avg`. Days with no provider data are omitted, but a
day that is present always carries all three provider keys, using `null` where a
provider had no data that day. Invalid `range` or `agg` values return `400`.

### `GET /api/analytics/breakdown`

Returns a flat `rows` table of every account × model with its current fraction,
reset time, and — when the current quota cycle has a distinct earlier snapshot —
its starting fraction and consumed delta. Rows lacking sufficient data omit
`starting_fraction` and `consumed` rather than guessing. Rows are returned
most-consumed first; the frontend re-sorts client-side.

### `GET /api/analytics/stats`

Returns four server-computed headline figures: total polls in the last seven
days, the most depleted model, the account with the most remaining quota, and
the soonest upcoming reset. Figures with no supporting data are `null`.

### `GET /api/accounts/{email}/sparklines`

Returns a per-model time series for the last seven days, one point per snapshot,
ascending by time. Null fractions are preserved so the frontend can render gaps.

### `GET /api/accounts/{email}/timeline`

Returns inferred login/logout events for the last seven days, ascending by time.
A gap between consecutive snapshots wider than the session-gap threshold closes
one session and opens the next. Each event carries a quota summary at that
boundary grouped by provider.

## Data Files

The daemon stores local runtime data under `~/.agq`:

- `agq.db`: SQLite database.
- `agq.log`: JSON log file.

The SQLite database uses WAL mode so dashboard readers can read while the daemon
writes.

## Development Notes

Keep package dependencies flowing inward:

- `cmd/agq-daemon` may import every internal package.
- `api`, `poller`, and `detector` depend on interfaces where practical.
- `domain` has no project-internal dependencies.
- `store` owns SQL details and returns `domain` types.
- `languageserver` owns Antigravity response parsing and HTTP request details.

Prefer adding tests at package boundaries. Existing coverage focuses on:

- Language server JSON parsing.
- `/proc/net/tcp` parsing and command-line flag extraction.
- Thread-safe status snapshots.
- SQLite persistence and queries.
- API error/status behavior.
- Poller deduplication and idle transitions.
