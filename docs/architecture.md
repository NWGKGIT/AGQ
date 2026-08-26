# Architecture

AGQ is one Go monitor core with two shells around it: a Wails desktop app and
an optional headless daemon. Both run the identical discovery → poll → persist
→ serve pipeline; they differ only in how the API handler is reached.

```
                 ┌────────────────────────────────────────────────┐
                 │                monitor.Runtime                 │
                 │                                                │
  /proc, WinAPI ─┼─▶ detector ──▶ poller ──▶ store (SQLite WAL)   │
                 │      │            │           ▲                │
                 │      └── probe ───┘           │                │
                 │      (GetUserStatus)      api.Server ──────────┼─▶ JSON
                 │                               ▲                │
                 └───────────────────────────────┼────────────────┘
                                                 │
             desktop app: in-process handler ────┤
             headless daemon: TCP 127.0.0.1:7432 ┘
```

## Package map

| Package                      | Responsibility                                                                                                 |
| ---------------------------- | -------------------------------------------------------------------------------------------------------------- |
| `cmd/agq-daemon`             | Headless entry point: logging, signals, wiring.                                                                |
| `monitor`                    | `Runtime` that owns and wires store, detector, poller, API server; embeddable.                                 |
| `internal/domain`            | Shared domain types. No project-internal dependencies.                                                         |
| `internal/detector`          | Process scanning (`/proc` on Linux, Toolhelp32/TcpTable on Windows), cmdline parsing, loopback port discovery. |
| `internal/languageserver`    | `GetUserStatus` HTTP client and response parsing.                                                              |
| `internal/poller`            | Poll scheduling, account dedup, persistence, daemon status updates.                                            |
| `internal/store`             | SQLite schema, migrations, writes, read queries.                                                               |
| `internal/state`             | Thread-safe daemon status snapshots.                                                                           |
| `internal/api`               | HTTP handlers, serve-time logic (assumed refill), CORS.                                                        |
| `desktop`                    | Wails shell: `App` bindings, config, in-process API client.                                                    |
| `desktop/internal/apiclient` | Typed client that dispatches requests straight to the embedded handler (no sockets).                           |
| `desktop/internal/config`    | `~/.agq/desktop.json` settings (port, API exposure, email masking).                                            |

Dependency rules: `domain` is imported by everything and imports nothing;
`store` owns SQL and returns `domain` types; `api`, `poller`, and `detector`
depend on interfaces where practical; `cmd` and `monitor` may import all.

## Runtime flow

1. The detector scans processes every **5 s** (`detector.DefaultScanInterval`)
   for Antigravity language servers (see [detection.md](detection.md)).
2. Detected processes are validated by an actual `GetUserStatus` probe; the
   first loopback port that answers with a valid email becomes `ActivePort`.
3. The poller polls the single authoritative process every **3 s**
   (`poller.DefaultNormalInterval`), deduplicates results by email (last
   success per cycle wins), and persists snapshots.
4. After **5** consecutive all-failure cycles the poll interval backs off to
   **5 min**; the first success restores 60 s.
5. `internal/state` tracks `ACTIVE`/`IDLE`, the active emails, and
   `last_poll_at` / `next_poll_at`.
6. `api.Server` serves everything read-only; serve-time logic like
   assumed-refill (see [data-and-logic.md](data-and-logic.md)) lives here so
   stored data is never rewritten.

## Desktop shell vs headless daemon

**Desktop (`desktop/`)** constructs `monitor.Runtime` with an empty `Addr`, so
no TCP listener is bound by default. The React frontend calls Wails-bound Go
methods (`GetAccounts`, `GetAccountModels`, …); these dispatch through
`apiclient.Client`, which synthesizes an `http.Request` and calls the
monitor's handler directly in-process (`httptest` recorder, no sockets). The
client resolves the handler per request, so it stays valid across monitor
restarts and reports "unreachable" while the monitor is down.

Enabling **Expose API** in Settings restarts the monitor with
`Addr=127.0.0.1:<port>` so external tools (curl, scripts) can read the same
API the UI uses.

**Headless daemon (`cmd/agq-daemon`)** is the same `Runtime` with the TCP
listener always on. The systemd user unit (`agq.service`) runs it on login
and appends JSON logs to `~/.agq/agq.log`.

Both shells share `~/.agq/agq.db`. SQLite runs in WAL mode so a dashboard can
read while a daemon writes - but run only one writer at a time (don't run the
desktop app and the daemon simultaneously against the same database).

## Lifecycle

`monitor.New` validates config; `Start` opens the database, binds the optional
listener, and launches scanner/poller/server goroutines; `Stop` shuts down
gracefully with a 5 s timeout. The desktop app calls these on Wails
startup/shutdown and when listener-affecting settings change.
